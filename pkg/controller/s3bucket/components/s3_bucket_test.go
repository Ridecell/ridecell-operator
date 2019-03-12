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

package components_test

import (
	"encoding/json"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pkg/errors"

	s3bucketcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/s3bucket/components"
)

type mockS3Client struct {
	s3iface.S3API
	mockBucketExists    bool
	mockBucketPolicy    *string
	mockBucketNameTaken bool
	mockBucketTagged    bool

	putPolicy        bool
	putPolicyContent string
	putBucketTagging bool
	deletePolicy     bool
}

var _ = Describe("s3bucket aws Component", func() {
	comp := s3bucketcomponents.NewS3Bucket()
	var mockS3 *mockS3Client

	BeforeEach(func() {
		comp = s3bucketcomponents.NewS3Bucket()
		mockS3 = &mockS3Client{}
		comp.InjectS3Factory(func(_ string) (s3iface.S3API, error) { return mockS3, nil })
	})

	It("runs basic reconcile with no existing bucket", func() {
		instance.Spec.BucketName = "foo-default-static"
		Expect(comp).To(ReconcileContext(ctx))
	})

	Describe("isReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("reconciles with existing bucket policy", func() {
		mockS3.mockBucketExists = true
		mockS3.mockBucketPolicy = aws.String(`
			{
				"Version": "2008-10-17",
				"Statement": []
			}
		`)
		mockS3.mockBucketTagged = true

		instance.Spec.BucketName = "foo-default-static"
		instance.Spec.BucketPolicy = `{
			"Version": "2008-10-17",
			"Statement": [{
				 "Sid": "PublicReadForGetBucketObjects",
				 "Effect": "Allow",
				 "Principal": {
					 "AWS": "*"
				 },
				 "Action": "s3:GetObject",
				 "Resource": "arn:aws:s3:::foo-default-static/*"
			 }]
		}`

		Expect(comp).To(ReconcileContext(ctx))

		Expect(mockS3.putPolicy).To(BeTrue())
		Expect(mockS3.putPolicyContent).To(MatchJSON(instance.Spec.BucketPolicy))
		Expect(mockS3.putBucketTagging).To(BeFalse())
	})

	It("adds a new bucket policy if not present", func() {
		mockS3.mockBucketExists = true

		instance.Spec.BucketName = "foo-default-static"
		instance.Spec.BucketPolicy = `{
			"Version": "2008-10-17",
			"Statement": [{
				 "Sid": "PublicReadForGetBucketObjects",
				 "Effect": "Allow",
				 "Principal": {
					 "AWS": "*"
				 },
				 "Action": "s3:GetObject",
				 "Resource": "arn:aws:s3:::foo-default-static/*"
			 }]
		}`

		Expect(comp).To(ReconcileContext(ctx))

		Expect(mockS3.putPolicy).To(BeTrue())
		Expect(mockS3.putPolicyContent).To(MatchJSON(instance.Spec.BucketPolicy))
		Expect(mockS3.putBucketTagging).To(BeTrue())
	})

	It("fails because bucket name is taken", func() {
		mockS3.mockBucketNameTaken = true

		instance.Spec.BucketName = "foo-default-static"

		Expect(comp).ToNot(ReconcileContext(ctx))
	})

	It("deletes a bucket policy if needed", func() {
		mockS3.mockBucketExists = true
		mockS3.mockBucketPolicy = aws.String(`
			{
				"Version": "2008-10-17",
				"Statement": []
			}
		`)

		Expect(comp).To(ReconcileContext(ctx))

		Expect(mockS3.putPolicy).To(BeFalse())
		Expect(mockS3.deletePolicy).To(BeTrue())
	})
})

// Mock aws functions below

func (m *mockS3Client) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	if m.mockBucketExists {
		return &s3.ListObjectsOutput{}, nil
	} else {
		return nil, awserr.New("NoSuchBucket", "", nil)
	}
}

func (m *mockS3Client) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	if aws.StringValue(input.Bucket) != instance.Spec.BucketName {
		return &s3.CreateBucketOutput{}, errors.New("awsmock_createbucket: bucket name was incorrect")
	}
	if aws.StringValue(input.CreateBucketConfiguration.LocationConstraint) != instance.Spec.Region {
		return &s3.CreateBucketOutput{}, errors.New("awsmock_createbucket: region was incorrect")
	}
	if m.mockBucketNameTaken {
		return &s3.CreateBucketOutput{}, errors.New("awsmock_createbucket: bucket name taken")
	}
	return &s3.CreateBucketOutput{}, nil
}

func (m *mockS3Client) GetBucketPolicy(input *s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
	if aws.StringValue(input.Bucket) != instance.Spec.BucketName {
		return &s3.GetBucketPolicyOutput{}, errors.New("awsmock_getbucketpolicy: bucketname was incorrect")
	}
	if m.mockBucketPolicy == nil {
		return nil, awserr.New("NoSuchBucketPolicy", "", nil)
	}
	return &s3.GetBucketPolicyOutput{Policy: m.mockBucketPolicy}, nil
}

func (m *mockS3Client) PutBucketPolicy(input *s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
	// Check bucket name.
	if aws.StringValue(input.Bucket) != instance.Spec.BucketName {
		return nil, awserr.New("NoSuchBucket", "", nil)
	}
	// Check that we have valid JSON.
	var ignored interface{}
	err := json.Unmarshal([]byte(*input.Policy), &ignored)
	if err != nil {
		return nil, awserr.New("InvalidPolicyDocument", "", nil)
	}
	m.putPolicy = true
	m.putPolicyContent = *input.Policy
	return &s3.PutBucketPolicyOutput{}, nil
}

func (m *mockS3Client) DeleteBucketPolicy(input *s3.DeleteBucketPolicyInput) (*s3.DeleteBucketPolicyOutput, error) {
	// Check bucket name.
	if aws.StringValue(input.Bucket) != instance.Spec.BucketName {
		return nil, awserr.New("NoSuchBucket", "", nil)
	}
	m.deletePolicy = true
	return &s3.DeleteBucketPolicyOutput{}, nil
}

func (m *mockS3Client) GetBucketTagging(input *s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	if aws.StringValue(input.Bucket) != instance.Spec.BucketName {
		return nil, awserr.New("NoSuchBucket", "", nil)
	}
	if m.mockBucketTagged {
		return &s3.GetBucketTaggingOutput{
			TagSet: []*s3.Tag{
				&s3.Tag{
					Key:   aws.String("ridecell-operator"),
					Value: aws.String("True"),
				},
			},
		}, nil
	}
	return &s3.GetBucketTaggingOutput{}, awserr.New("NoSuchTagSet", "", nil)
}

func (m *mockS3Client) PutBucketTagging(input *s3.PutBucketTaggingInput) (*s3.PutBucketTaggingOutput, error) {
	if aws.StringValue(input.Bucket) != instance.Spec.BucketName {
		return nil, awserr.New("NoSuchBucket", "", nil)
	}
	m.putBucketTagging = true
	return &s3.PutBucketTaggingOutput{}, nil
}
