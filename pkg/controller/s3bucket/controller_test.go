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
	"fmt"
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sess *session.Session
var s3svc *s3.S3
var s3Bucket *awsv1beta1.S3Bucket
var randOwnerPrefix string

var _ = Describe("s3bucket controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping s3bucket integration tests")
		}

		randOwnerPrefix = os.Getenv("RAND_OWNER_PREFIX")
		if randOwnerPrefix == "" {
			panic("$RAND_OWNER_PREFIX not set, failing test")
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
			panic("These tests should only be run on the testing account.")
		}

		s3svc = s3.New(sess)

		s3Bucket = &awsv1beta1.S3Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
			Spec: awsv1beta1.S3BucketSpec{
				Region: "us-west-2",
			},
		}
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient
		bucketName := fmt.Sprintf("ridecell-%s-s3bucket-test-static", randOwnerPrefix)
		s3Bucket.Spec.BucketName = bucketName
		s3Bucket.Spec.BucketPolicy = fmt.Sprintf(`{
			"Version": "2008-10-17",
			"Statement": [{
				 "Sid": "PublicReadForGetBucketObjects",
				 "Effect": "Allow",
				 "Principal": {
					 "AWS": "*"
				 },
				 "Action": "s3:GetObject",
				 "Resource": "arn:aws:s3:::%s/*"
			 }]
		}`, bucketName)
		c.Create(s3Bucket)

		fetchBucket := &awsv1beta1.S3Bucket{}
		c.EventuallyGet(helpers.Name("test"), fetchBucket, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(bucketExists()).ToNot(HaveOccurred())
		Expect(bucketHasValidTag()).ToNot(HaveOccurred())
		Expect(getBucketPolicy()).To(MatchJSON(s3Bucket.Spec.BucketPolicy))
	})

	It("has an invalid bucket policy", func() {
		c := helpers.TestClient
		bucketName := fmt.Sprintf("ridecell-%s-invalidpolicy-test-static", randOwnerPrefix)
		s3Bucket.Spec.BucketName = bucketName
		s3Bucket.Spec.BucketPolicy = "invalid"
		c.Create(s3Bucket)

		fetchBucket := &awsv1beta1.S3Bucket{}
		c.EventuallyGet(helpers.Name("test"), fetchBucket, c.EventuallyStatus(awsv1beta1.StatusError))

		Expect(bucketExists()).ToNot(HaveOccurred())
		Expect(bucketHasValidTag()).ToNot(HaveOccurred())
	})

	It("finds a bucket that already exists", func() {
		c := helpers.TestClient
		bucketName := fmt.Sprintf("ridecell-%s-preexisting-test-static", randOwnerPrefix)
		s3Bucket.Spec.BucketName = bucketName
		s3Bucket.Spec.BucketPolicy = fmt.Sprintf(`{
			"Version": "2008-10-17",
			"Statement": [{
				 "Sid": "PublicReadForGetBucketObjects",
				 "Effect": "Allow",
				 "Principal": {
					 "AWS": "*"
				 },
				 "Action": "s3:GetObject",
				 "Resource": "arn:aws:s3:::%s/*"
			 }]
		}`, bucketName)
		_, err := s3svc.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(bucketName)})
		Expect(err).ToNot(HaveOccurred())
		c.Create(s3Bucket)

		fetchBucket := &awsv1beta1.S3Bucket{}
		c.EventuallyGet(helpers.Name("test"), fetchBucket, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(bucketExists()).ToNot(HaveOccurred())
		Expect(bucketHasValidTag()).ToNot(HaveOccurred())
		Expect(getBucketPolicy()).To(MatchJSON(s3Bucket.Spec.BucketPolicy))
	})

	It("updates existing bucket policy", func() {
		bucketName := fmt.Sprintf("ridecell-%s-mismatchpolicy-test-static", randOwnerPrefix)
		_, err := s3svc.CreateBucket(&s3.CreateBucketInput{Bucket: aws.String(bucketName)})
		Expect(err).ToNot(HaveOccurred())

		oldPolicy := fmt.Sprintf(`{
			"Version": "2008-10-17",
			"Statement": [{
				 "Sid": "PublicReadForGetBucketObjects",
				 "Effect": "Deny",
				 "Principal": {
					 "AWS": "*"
				 },
				 "Action": "s3:GetObject",
				 "Resource": "arn:aws:s3:::%s/*"
			 }]
		}`, bucketName)

		_, err = s3svc.PutBucketPolicy(&s3.PutBucketPolicyInput{
			Bucket: aws.String(bucketName),
			Policy: aws.String(oldPolicy),
		})
		Expect(err).ToNot(HaveOccurred())

		c := helpers.TestClient
		s3Bucket.Spec.BucketName = bucketName
		s3Bucket.Spec.BucketPolicy = fmt.Sprintf(`{
			"Version": "2008-10-17",
			"Statement": [{
				 "Sid": "PublicReadForGetBucketObjects",
				 "Effect": "Allow",
				 "Principal": {
					 "AWS": "*"
				 },
				 "Action": "s3:GetObject",
				 "Resource": "arn:aws:s3:::%s/*"
			 }]
		}`, bucketName)

		c.Create(s3Bucket)

		fetchBucket := &awsv1beta1.S3Bucket{}
		c.EventuallyGet(helpers.Name("test"), fetchBucket, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(bucketExists()).ToNot(HaveOccurred())
		Expect(bucketHasValidTag()).ToNot(HaveOccurred())
		Expect(getBucketPolicy()).To(MatchJSON(s3Bucket.Spec.BucketPolicy))
	})

	It("Has a blank BucketPolicy in spec", func() {
		c := helpers.TestClient
		bucketName := fmt.Sprintf("ridecell-%s-blankpolicy-test-static", randOwnerPrefix)
		s3Bucket.Spec.BucketName = bucketName
		c.Create(s3Bucket)

		fetchBucket := &awsv1beta1.S3Bucket{}
		c.EventuallyGet(helpers.Name("test"), fetchBucket, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(bucketExists()).ToNot(HaveOccurred())
		Expect(bucketHasValidTag()).ToNot(HaveOccurred())

		_, err := s3svc.GetBucketPolicy(&s3.GetBucketPolicyInput{Bucket: aws.String(bucketName)})
		Expect(err).To(HaveOccurred())
	})
})

func bucketExists() error {
	_, err := s3svc.ListObjects(&s3.ListObjectsInput{
		Bucket:  aws.String(s3Bucket.Spec.BucketName),
		MaxKeys: aws.Int64(1),
	})
	return err
}

func bucketHasValidTag() error {
	getBucketTags, err := s3svc.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String(s3Bucket.Spec.BucketName)})
	if err != nil {
		return err
	}
	for _, tagSet := range getBucketTags.TagSet {
		if aws.StringValue(tagSet.Key) == "ridecell-operator" {
			return nil
		}
	}
	return errors.New("did not find ridecell-operator bucket tag")
}

func getBucketPolicy() string {
	getBucketPolicyObj, err := s3svc.GetBucketPolicy(&s3.GetBucketPolicyInput{Bucket: aws.String(s3Bucket.Spec.BucketName)})
	if err != nil {
		return ""
	}
	return aws.StringValue(getBucketPolicyObj.Policy)
}
