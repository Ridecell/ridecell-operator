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

package s3bucket_test

import (
	"context"
	"encoding/json"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
	"os"
	"reflect"
	"time"

	awssv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const timeout = time.Second * 30

var _ = Describe("s3bucket controller", func() {
	var helpers *test_helpers.PerTestHelpers

	var s3svc *s3.S3
	var sess *session.Session

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping s3bucket integration tests")
		}
		var err error
		sess, err = session.NewSession(&aws.Config{
			Region: aws.String("us-west-2"),
		})
		Expect(err).NotTo(HaveOccurred())

		// Check if this being run on the testing account
		stssvc := sts.New(sess)
		getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		Expect(err).NotTo(HaveOccurred())
		if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
			Skip("These tests should only be run on the testing account.")
		}

		s3svc = s3.New(sess)
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("runs a full reconcile", func() {
		s3Bucket := &awssv1beta1.S3Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
			Spec: awssv1beta1.S3BucketSpec{
				BucketName: "ridecell-s3bucket-test-static",
				Region:     "us-west-2",
				BucketPolicy: `
		    {
		      "Version": "2008-10-17",
		      "Statement": [{
		         "Sid": "PublicReadForGetBucketObjects",
		         "Effect": "Allow",
		         "Principal": {
		           "AWS": "*"
		         },
		         "Action": "s3:GetObject",
		         "Resource": "arn:aws:s3:::ridecell-s3bucket-test-static/*"
		       }]
		    }`,
			},
		}

		err := helpers.Client.Create(context.TODO(), s3Bucket)
		Expect(err).ToNot(HaveOccurred())

		// If bucket doesn't exist this will error
		Eventually(func() error {
			_, err = s3svc.ListObjects(&s3.ListObjectsInput{
				Bucket:  aws.String("ridecell-s3bucket-test-static"),
				MaxKeys: aws.Int64(1),
			})
			return err
		}, timeout).Should(Succeed())

		// Make sure our bucket has the correct tags
		Eventually(func() error {
			getBucketTags, err := s3svc.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String("ridecell-s3bucket-test-static")})
			if err != nil {
				return err
			}
			for _, tagSet := range getBucketTags.TagSet {
				if aws.StringValue(tagSet.Key) == "ridecell-operator" {
					return nil
				}
			}
			return errors.New("did not find ridecell-operator bucket tag")
		}, timeout).Should(Succeed())

		// Match our bucket policy
		Eventually(func() error {
			getBucketPolicyObj, err := s3svc.GetBucketPolicy(&s3.GetBucketPolicyInput{Bucket: aws.String("ridecell-s3bucket-test-static")})
			if err != nil {
				return err
			}
			var existingPolicy interface{}
			var goalPolicy interface{}
			err = json.Unmarshal([]byte(*getBucketPolicyObj.Policy), &existingPolicy)
			if err != nil {
				return err
			}
			err = json.Unmarshal([]byte(s3Bucket.Spec.BucketPolicy), &goalPolicy)
			if err != nil {
				return err
			}
			if reflect.DeepEqual(existingPolicy, goalPolicy) {
				return nil
			}
			return errors.New("Bucket policies did not match")
		}, timeout).Should(Succeed())
	})

})
