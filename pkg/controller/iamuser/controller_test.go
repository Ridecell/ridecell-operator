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

package iamuser_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sess *session.Session
var iamsvc *iam.IAM
var iamUser *awsv1beta1.IAMUser
var randOwnerPrefix string

var _ = Describe("iamuser controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping iamuser integration tests")
		}
		if os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN") == "" {
			Skip("$AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN not set, skipping iamuser integration tests")
		}

		randOwnerPrefix = os.Getenv("RAND_OWNER_PREFIX")
		if randOwnerPrefix == "" {
			panic("$RAND_OWNER_PREFIX not set, failing test")
		}

		var err error
		sess, err = session.NewSession()
		Expect(err).NotTo(HaveOccurred())

		// Check if this being run on the testing account
		stssvc := sts.New(sess)
		getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		Expect(err).NotTo(HaveOccurred())
		if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
			Skip("These tests should only be run on the testing account.")
		}

		iamsvc = iam.New(sess)

		iamUser = &awsv1beta1.IAMUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
			Spec: awsv1beta1.IAMUserSpec{
				InlinePolicies: map[string]string{
					"allow_s3": `{
						 "Version": "2012-10-17",
						 "Statement": {
							 "Effect": "Allow",
							 "Action": [
									"s3:ListBucket",
									"s3:GetObject",
									"s3:DeleteObject",
									"s3:PutObject"
								],
							 "Resource": "arn:aws:s3:::ridecell-invalid-static*"
						 }
					}`,
					"allow_sqs": `{
						"Version": "2012-10-17",
						"Statement": {
							"Sid": "",
							"Effect": "Allow",
							"Action": [
								"sqs:SendMessageBatch",
								"sqs:SendMessage",
								"sqs:CreateQueue"
							],
							"Resource": "arn:aws:sqs:us-west-2::invalid"
						}
					}`,
				},
				PermissionsBoundaryArn: os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN"),
			},
		}
	})

	AfterEach(func() {
		// Delete user and see if it cleans up on its own
		c := helpers.TestClient
		c.Delete(iamUser)
		Eventually(func() error { return userExists() }, time.Second*10).ShouldNot(Succeed())

		// Make sure the object is deleted
		fetchIAMUser := &awsv1beta1.IAMUser{}
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name(iamUser.Name), fetchIAMUser)
		}, time.Second*30).ShouldNot(Succeed())

		helpers.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient
		iamUser.Spec.UserName = fmt.Sprintf("%s-basicuser-test-summon-platform", randOwnerPrefix)
		c.Create(iamUser)

		fetchIAMUser := &awsv1beta1.IAMUser{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMUser, c.EventuallyStatus(awsv1beta1.StatusReady))

		fetchAccessKey := &corev1.Secret{}
		c.Get(helpers.Name("test.aws-credentials"), fetchAccessKey)

		Expect(aws.StringValue(getAccessKeys()[0].AccessKeyId)).To(Equal(string(fetchAccessKey.Data["AWS_ACCESS_KEY_ID"])))
		Expect(userExists()).ToNot(HaveOccurred())
		Expect(userHasValidTag()).To(BeTrue())
		Expect(getUserPolicyNames()).To(HaveLen(2))
		Expect(getUserPolicyDocument("allow_s3")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_s3"]))
		Expect(getUserPolicyDocument("allow_sqs")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_sqs"]))
		Expect(getAccessKeys()).To(HaveLen(1))

		Expect(fetchIAMUser.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMUser.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("deletes old access key that does not match secret", func() {
		c := helpers.TestClient
		username := fmt.Sprintf("%s-preexistinguser-test-summon-platform", randOwnerPrefix)
		_, err := iamsvc.CreateUser(&iam.CreateUserInput{
			UserName:            aws.String(username),
			PermissionsBoundary: aws.String(iamUser.Spec.PermissionsBoundaryArn),
		})
		Expect(err).ToNot(HaveOccurred())

		_, err = iamsvc.CreateAccessKey(&iam.CreateAccessKeyInput{
			UserName: aws.String(username),
		})
		Expect(err).ToNot(HaveOccurred())

		iamUser.Spec.UserName = username
		c.Create(iamUser)

		fetchIAMUser := &awsv1beta1.IAMUser{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMUser, c.EventuallyStatus(awsv1beta1.StatusReady))

		fetchAccessKey := &corev1.Secret{}
		c.Get(helpers.Name("test.aws-credentials"), fetchAccessKey)

		Expect(aws.StringValue(getAccessKeys()[0].AccessKeyId)).To(Equal(string(fetchAccessKey.Data["AWS_ACCESS_KEY_ID"])))
		Expect(userExists()).ToNot(HaveOccurred())
		Expect(userHasValidTag()).To(BeTrue())
		Expect(getUserPolicyNames()).To(HaveLen(2))
		Expect(getUserPolicyDocument("allow_s3")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_s3"]))
		Expect(getUserPolicyDocument("allow_sqs")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_sqs"]))
		Expect(getAccessKeys()).To(HaveLen(1))

		Expect(fetchIAMUser.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMUser.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("deletes existing user policies not in spec", func() {
		c := helpers.TestClient
		username := fmt.Sprintf("%s-policynotinspec-test-summon-platform", randOwnerPrefix)
		_, err := iamsvc.CreateUser(&iam.CreateUserInput{
			UserName:            aws.String(username),
			PermissionsBoundary: aws.String(iamUser.Spec.PermissionsBoundaryArn),
		})
		Expect(err).ToNot(HaveOccurred())

		existingPolicy := `{
			"Version": "2012-10-17",
			"Statement": {
				"Sid": "",
				"Effect": "Deny",
				"Action": "s3:*",
				"Resource": "*"
			}
		}`
		_, err = iamsvc.PutUserPolicy(&iam.PutUserPolicyInput{
			PolicyDocument: aws.String(existingPolicy),
			PolicyName:     aws.String("existing_policy"),
			UserName:       aws.String(username),
		})
		Expect(err).ToNot(HaveOccurred())

		iamUser.Spec.UserName = username
		c.Create(iamUser)

		fetchIAMUser := &awsv1beta1.IAMUser{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMUser, c.EventuallyStatus(awsv1beta1.StatusReady))

		fetchAccessKey := &corev1.Secret{}
		c.Get(helpers.Name("test.aws-credentials"), fetchAccessKey)

		Expect(aws.StringValue(getAccessKeys()[0].AccessKeyId)).To(Equal(string(fetchAccessKey.Data["AWS_ACCESS_KEY_ID"])))
		Expect(userExists()).ToNot(HaveOccurred())
		Expect(userHasValidTag()).To(BeTrue())
		Expect(getUserPolicyNames()).To(HaveLen(2))
		Expect(getUserPolicyDocument("allow_s3")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_s3"]))
		Expect(getUserPolicyDocument("allow_sqs")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_sqs"]))
		Expect(getAccessKeys()).To(HaveLen(1))

		Expect(fetchIAMUser.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMUser.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("fails to create user with bad inlinepolicies json", func() {
		c := helpers.TestClient
		username := fmt.Sprintf("%s-badpolicyjson-test-summon-platform", randOwnerPrefix)
		iamUser.Spec.UserName = username
		iamUser.Spec.InlinePolicies["allow_s3"] = "invalid"
		iamUser.Spec.InlinePolicies["allow_sqs"] = "invalid"
		c.Create(iamUser)

		fetchIAMUser := &awsv1beta1.IAMUser{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMUser, c.EventuallyStatus(awsv1beta1.StatusError))

		Expect(fetchIAMUser.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMUser.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("ensures that object isn't deleted prematurely by finalizer", func() {
		c := helpers.TestClient
		username := fmt.Sprintf("%s-prematuredelete-test-summon-platform", randOwnerPrefix)
		iamUser.Spec.UserName = username
		c.Create(iamUser)

		fetchIAMUser := &awsv1beta1.IAMUser{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMUser, c.EventuallyStatus(awsv1beta1.StatusReady))

		fetchAccessKey := &corev1.Secret{}
		c.Get(helpers.Name("test.aws-credentials"), fetchAccessKey)

		Expect(aws.StringValue(getAccessKeys()[0].AccessKeyId)).To(Equal(string(fetchAccessKey.Data["AWS_ACCESS_KEY_ID"])))
		Expect(userExists()).ToNot(HaveOccurred())
		Expect(userHasValidTag()).To(BeTrue())
		Expect(getUserPolicyNames()).To(HaveLen(2))
		Expect(getUserPolicyDocument("allow_s3")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_s3"]))
		Expect(getUserPolicyDocument("allow_sqs")).To(MatchJSON(iamUser.Spec.InlinePolicies["allow_sqs"]))
		Expect(getAccessKeys()).To(HaveLen(1))

		userAccessKeys := getAccessKeys()
		Expect(aws.StringValue(userAccessKeys[0].AccessKeyId)).To(Equal(string(fetchAccessKey.Data["AWS_ACCESS_KEY_ID"])))

		Consistently(func() error { return userExists() }, time.Second*20).Should(Succeed())

		Expect(fetchIAMUser.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMUser.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})
})

func userExists() error {
	_, err := iamsvc.GetUser(&iam.GetUserInput{UserName: aws.String(iamUser.Spec.UserName)})
	return err
}

func userHasValidTag() bool {
	listUserTagsOutput, err := iamsvc.ListUserTags(&iam.ListUserTagsInput{UserName: aws.String(iamUser.Spec.UserName)})
	if err != nil {
		return false
	}

	for _, TagSet := range listUserTagsOutput.Tags {
		if aws.StringValue(TagSet.Key) == "ridecell-operator" {
			return true
		}
	}
	return false
}

func getAccessKeys() []*iam.AccessKeyMetadata {
	listAccessKeysOutput, err := iamsvc.ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(iamUser.Spec.UserName)})
	if err != nil {
		return nil
	}
	return listAccessKeysOutput.AccessKeyMetadata
}

func getUserPolicyNames() []*string {
	getUserPoliciesOutput, err := iamsvc.ListUserPolicies(&iam.ListUserPoliciesInput{UserName: aws.String(iamUser.Spec.UserName)})
	if err != nil {
		return nil
	}
	return getUserPoliciesOutput.PolicyNames
}

func getUserPolicyDocument(userPolicyName string) string {
	getUserPolicy, err := iamsvc.GetUserPolicy(&iam.GetUserPolicyInput{
		UserName:   aws.String(iamUser.Spec.UserName),
		PolicyName: aws.String(userPolicyName),
	})
	if err != nil {
		return ""
	}

	decoded, err := url.PathUnescape(aws.StringValue(getUserPolicy.PolicyDocument))
	if err != nil {
		return ""
	}
	return decoded
}
