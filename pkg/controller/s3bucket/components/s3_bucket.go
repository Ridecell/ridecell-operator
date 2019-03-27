/*
Copyright 2019 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package components

import (
	"encoding/json"
	"reflect"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
)

const s3BucketFinalizer = "s3bucket.finalizer"

type S3Factory func(region string) (s3iface.S3API, error)

type s3BucketComponent struct {
	// Keep an S3API per region.
	s3Services map[string]s3iface.S3API
	s3Factory  S3Factory
}

func realS3Factory(region string) (s3iface.S3API, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}
	return s3.New(sess), nil
}

func NewS3Bucket() *s3BucketComponent {
	return &s3BucketComponent{
		s3Services: map[string]s3iface.S3API{},
		s3Factory:  realS3Factory,
	}
}

func (comp *s3BucketComponent) InjectS3Factory(factory S3Factory) {
	comp.s3Factory = factory
}

func (_ *s3BucketComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *s3BucketComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *s3BucketComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.S3Bucket)

	// if object is not being deleted
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Is our finalizer attached to the object?
		if !containsString(s3BucketFinalizer, instance.ObjectMeta.Finalizers) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, s3BucketFinalizer)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{Requeue: true}, errors.Wrapf(err, "s3bucket: failed to update instance while adding finalizer")
			}
			return components.Result{Requeue: true}, nil
		}
	} else {
		if containsString(s3BucketFinalizer, instance.ObjectMeta.Finalizers) {
			result, err := comp.deleteDependencies(ctx)
			if err != nil {
				return result, err
			}
			// All operations complete, remove finalizer
			removeString(s3BucketFinalizer, instance.ObjectMeta.Finalizers)
			err = ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{Requeue: true}, errors.Wrapf(err, "s3bucket: failed to update instance while removing finalizer")
			}
			return components.Result{}, nil
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// Get an S3 API to work with. This has to match the bucket region.
	s3Service, err := comp.getS3(instance)
	if err != nil {
		return components.Result{}, err
	}

	// Run a ListBucket call to check if this bucket exists.
	bucketExists := true
	_, err = s3Service.ListObjects(&s3.ListObjectsInput{
		Bucket:  aws.String(instance.Spec.BucketName),
		MaxKeys: aws.Int64(1), // We don't actually care about the keys, so set this down for perf.
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchBucket {
			bucketExists = false
		} else {
			return components.Result{}, errors.Wrapf(err, "s3_bucket: error listing objects in %s", instance.Spec.BucketName)
		}
	}

	// If the bucket does not exist create it
	if !bucketExists {
		_, err = s3Service.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(instance.Spec.BucketName),
			CreateBucketConfiguration: &s3.CreateBucketConfiguration{
				LocationConstraint: aws.String(instance.Spec.Region),
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "s3_bucket: failed to create bucket %s", instance.Spec.BucketName)
		}
	}

	// Look for ridecell-operator tag, if it doesn't exist create it
	getBucketTags, err := s3Service.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String(instance.Spec.BucketName)})
	if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "NoSuchTagSet" {
		// There is no tag set associated with the bucket.
		getBucketTags = &s3.GetBucketTaggingOutput{TagSet: []*s3.Tag{}}
	} else if err != nil {
		return components.Result{}, errors.Wrapf(err, "s3_bucket: failed to get bucket tags")
	}

	var foundTag bool
	for _, tagSet := range getBucketTags.TagSet {
		if aws.StringValue(tagSet.Key) == "ridecell-operator" {
			foundTag = true
		}
	}
	if !foundTag {
		_, err := s3Service.PutBucketTagging(&s3.PutBucketTaggingInput{
			Bucket: aws.String(instance.Spec.BucketName),
			Tagging: &s3.Tagging{
				TagSet: []*s3.Tag{
					&s3.Tag{
						Key:   aws.String("ridecell-operator"),
						Value: aws.String("True"),
					},
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "s3_bucket: failed to put bucket tags")
		}
	}

	// Try to grab the existing bucket policy.
	bucketHasPolicy := true
	getBucketPolicyObj, err := s3Service.GetBucketPolicy(&s3.GetBucketPolicyInput{Bucket: aws.String(instance.Spec.BucketName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NoSuchBucketPolicy" {
			bucketHasPolicy = false
		} else {
			return components.Result{}, errors.Wrapf(err, "s3_bucket: failed to get bucket policy for bucket %s", instance.Spec.BucketName)
		}
	}

	// If the policy is "", we need to delete if set. Otherwise we need to check for == and then put.
	if instance.Spec.BucketPolicy == "" {
		if bucketHasPolicy {
			_, err := s3Service.DeleteBucketPolicy(&s3.DeleteBucketPolicyInput{
				Bucket: aws.String(instance.Spec.BucketName),
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "s3_bucket: failed to delete bucket policy for bucket %s", instance.Spec.BucketName)
			}
		}
	} else {
		// Work out if we need to update the policy.
		bucketPolicyNeedsUpdate := false
		if bucketHasPolicy {
			var existingPolicy interface{}
			var goalPolicy interface{}
			err = json.Unmarshal([]byte(*getBucketPolicyObj.Policy), &existingPolicy)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "s3_bucket: error decoding existing bucket policy for bucket %s", instance.Spec.BucketName)
			}
			err = json.Unmarshal([]byte(instance.Spec.BucketPolicy), &goalPolicy)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "s3_bucket: error decoding goal bucket policy for bucket %s", instance.Spec.BucketName)
			}
			bucketPolicyNeedsUpdate = !reflect.DeepEqual(existingPolicy, goalPolicy)
		} else {
			// No existing policy, definitely update things.
			bucketPolicyNeedsUpdate = true
		}

		// Update or create the bucket policy.
		if bucketPolicyNeedsUpdate {
			_, err := s3Service.PutBucketPolicy(&s3.PutBucketPolicyInput{
				Bucket: aws.String(instance.Spec.BucketName),
				Policy: aws.String(instance.Spec.BucketPolicy),
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "s3_bucket: failed to put bucket policy for bucket %s", instance.Spec.BucketName)
			}
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*awsv1beta1.S3Bucket)
		instance.Status.Status = awsv1beta1.StatusReady
		instance.Status.Message = "Bucket exists and has correct policy"
		return nil
	}}, nil
}

func (comp *s3BucketComponent) getS3(instance *awsv1beta1.S3Bucket) (s3iface.S3API, error) {
	s3Service, ok := comp.s3Services[instance.Spec.Region]
	if ok {
		// Already open.
		return s3Service, nil
	}
	// Open a new session for this region.
	s3Service, err := comp.s3Factory(instance.Spec.Region)
	if err != nil {
		return nil, errors.Wrapf(err, "s3_bucket: error getting an S3 session for region %s", instance.Spec.Region)
	}
	comp.s3Services[instance.Spec.Region] = s3Service
	return s3Service, nil
}

func (comp *s3BucketComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.S3Bucket)
	s3Service, err := comp.getS3(instance)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "s3bucket: failed to get s3 client for finalizer")
	}

	// All objects in the bucket must be deleted prior to bucket deletion
	listObjectsOutput := []*s3.Object{}
	err = s3Service.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(instance.Spec.BucketName),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		listObjectsOutput = append(listObjectsOutput, page.Contents...)
		// If all results are returned IsTruncated returns false and breaks the loop.
		return aws.BoolValue(page.IsTruncated)
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != s3.ErrCodeNoSuchBucket {
			return components.Result{}, errors.Wrapf(err, "s3bucket: failed to get objects for finalizer")
		}
	}

	for _, s3Object := range listObjectsOutput {
		_, err := s3Service.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(instance.Spec.BucketName),
			Key:    s3Object.Key,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "s3bucket: failed to delete s3 object for finalizer")
		}
	}

	_, err = s3Service.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(instance.Spec.BucketName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != s3.ErrCodeNoSuchBucket {
			return components.Result{}, errors.Wrapf(aerr, "s3bucket: failed to delete bucket for finalizer")
		}
	}
	return components.Result{}, nil
}

func containsString(input string, slice []string) bool {
	for _, i := range slice {
		if i == input {
			return true
		}
	}
	return false
}

func removeString(input string, slice []string) []string {
	var outputSlice []string
	for _, i := range slice {
		if i == input {
			continue
		}
		outputSlice = append(outputSlice, input)
	}
	return outputSlice
}
