/*
Copyright 2020 Ridecell, Inc.

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

package iamrole_test

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sess *session.Session
var iamsvc *iam.IAM
var iamRole *awsv1beta1.IAMRole
var randOwnerPrefix string
var expectedRoleName string

var _ = Describe("iamrole controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		os.Setenv("ENABLE_FINALIZERS", "true")
		helpers = testHelpers.SetupTest()
		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping iamrole integration tests")
		}
		if os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN") == "" {
			Skip("$AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN not set, skipping iamrole integration tests")
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

		iamRole = &awsv1beta1.IAMRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
			Spec: awsv1beta1.IAMRoleSpec{
				AssumeRolePolicyDocument: `{ "Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Principal": {"Service": "ec2.amazonaws.com"}, "Action": "sts:AssumeRole"}]}`,
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
							"Resource": "arn:aws:sqs:us-west-1::invalid"
						}
					}`,
				},
				PermissionsBoundaryArn: os.Getenv("AWS_TEST_ACCOUNT_PERMISSIONS_BOUNDARY_ARN"),
			},
		}
		expectedRoleName = ""
	})

	AfterEach(func() {
		// Delete role and see if it cleans up on its own
		c := helpers.TestClient

		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&awsv1beta1.IAMRoleList{})
		}

		c.Delete(iamRole)
		if expectedRoleName == "" {
			expectedRoleName = iamRole.Spec.RoleName
		}
		Eventually(func() error { return roleExists(expectedRoleName) }, time.Second*10).ShouldNot(Succeed())

		// Make sure the object is deleted
		fetchIAMRole := &awsv1beta1.IAMRole{}
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name(iamRole.Name), fetchIAMRole)
		}, time.Second*30).ShouldNot(Succeed())

		helpers.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient
		iamRole.Spec.RoleName = fmt.Sprintf("%s-basicrole-test-summon-platform", randOwnerPrefix)
		c.Create(iamRole)

		fetchIAMRole := &awsv1beta1.IAMRole{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMRole, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(roleExists(iamRole.Spec.RoleName)).ToNot(HaveOccurred())
		Expect(roleHasValidTag(iamRole.Spec.RoleName)).To(BeTrue())
		Expect(getRolePolicyNames(iamRole.Spec.RoleName)).To(HaveLen(2))
		Expect(getRolePolicyDocument(iamRole.Spec.RoleName, "allow_s3")).To(MatchJSON(iamRole.Spec.InlinePolicies["allow_s3"]))
		Expect(getRolePolicyDocument(iamRole.Spec.RoleName, "allow_sqs")).To(MatchJSON(iamRole.Spec.InlinePolicies["allow_sqs"]))

		Expect(fetchIAMRole.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMRole.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("deletes existing role policies not in spec", func() {
		c := helpers.TestClient
		rolename := fmt.Sprintf("%s-policynotinspec-test-summon-platform", randOwnerPrefix)
		_, err := iamsvc.CreateRole(&iam.CreateRoleInput{
			RoleName:                 aws.String(rolename),
			PermissionsBoundary:      aws.String(iamRole.Spec.PermissionsBoundaryArn),
			AssumeRolePolicyDocument: aws.String(iamRole.Spec.AssumeRolePolicyDocument),
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
		_, err = iamsvc.PutRolePolicy(&iam.PutRolePolicyInput{
			PolicyDocument: aws.String(existingPolicy),
			PolicyName:     aws.String("existing_policy"),
			RoleName:       aws.String(rolename),
		})
		Expect(err).ToNot(HaveOccurred())

		iamRole.Spec.RoleName = rolename
		c.Create(iamRole)

		fetchIAMRole := &awsv1beta1.IAMRole{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMRole, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(roleExists(iamRole.Spec.RoleName)).ToNot(HaveOccurred())
		Expect(roleHasValidTag(iamRole.Spec.RoleName)).To(BeTrue())
		Expect(getRolePolicyNames(iamRole.Spec.RoleName)).To(HaveLen(2))
		Expect(getRolePolicyDocument(iamRole.Spec.RoleName, "allow_s3")).To(MatchJSON(iamRole.Spec.InlinePolicies["allow_s3"]))
		Expect(getRolePolicyDocument(iamRole.Spec.RoleName, "allow_sqs")).To(MatchJSON(iamRole.Spec.InlinePolicies["allow_sqs"]))

		Expect(fetchIAMRole.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMRole.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("fails to create role with bad inlinepolicies json", func() {
		c := helpers.TestClient
		rolename := fmt.Sprintf("%s-badpolicyjson-test-summon-platform", randOwnerPrefix)
		iamRole.Spec.RoleName = rolename
		iamRole.Spec.InlinePolicies["allow_s3"] = "invalid"
		iamRole.Spec.InlinePolicies["allow_sqs"] = "invalid"
		c.Create(iamRole)

		fetchIAMRole := &awsv1beta1.IAMRole{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMRole, c.EventuallyStatus(awsv1beta1.StatusError))

		Expect(fetchIAMRole.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMRole.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("ensures that object isn't deleted prematurely by finalizer", func() {
		c := helpers.TestClient
		rolename := fmt.Sprintf("%s-prematuredelete-test-summon-platform", randOwnerPrefix)
		iamRole.Spec.RoleName = rolename
		c.Create(iamRole)

		fetchIAMRole := &awsv1beta1.IAMRole{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMRole, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(roleExists(iamRole.Spec.RoleName)).ToNot(HaveOccurred())
		Expect(roleHasValidTag(iamRole.Spec.RoleName)).To(BeTrue())
		Expect(getRolePolicyNames(iamRole.Spec.RoleName)).To(HaveLen(2))
		Expect(getRolePolicyDocument(iamRole.Spec.RoleName, "allow_s3")).To(MatchJSON(iamRole.Spec.InlinePolicies["allow_s3"]))
		Expect(getRolePolicyDocument(iamRole.Spec.RoleName, "allow_sqs")).To(MatchJSON(iamRole.Spec.InlinePolicies["allow_sqs"]))

		Consistently(func() error { return roleExists(iamRole.Spec.RoleName) }, time.Second*20).Should(Succeed())

		Expect(fetchIAMRole.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMRole.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("uses templating functionality on every possible field", func() {
		c := helpers.TestClient
		iamRole.Spec.RoleName = fmt.Sprintf("%s-templating-test-{{ .Region }}-summon-platform", randOwnerPrefix)
		iamRole.Spec.AssumeRolePolicyDocument = fmt.Sprintf(`{"Version": "2012-10-17", "Statement": [{"Effect": "Allow","Principal": {"AWS": "arn:aws:iam::%s:role/iamrole-testing-role-{{ .Region }}"},"Action": "sts:AssumeRole"}]}`, os.Getenv("AWS_TESTING_ACCOUNT_ID"))
		iamRole.Spec.InlinePolicies = map[string]string{
			"test_tmpl": `{"Version": "2012-10-17", "Statement": {"Effect": "Deny", "Action": "s3:*", "Resource": "arn:aws:s3:::random-test-bucket-{{ .Region }}"}}`,
		}
		c.Create(iamRole)

		expectedRoleName = fmt.Sprintf("%s-templating-test-us-west-1-summon-platform", randOwnerPrefix)
		expectedAssumePolicy := fmt.Sprintf(`{"Version": "2012-10-17", "Statement": [{"Effect": "Allow","Principal": {"AWS": "arn:aws:iam::%s:role/iamrole-testing-role-us-west-1"},"Action": "sts:AssumeRole"}]}`, os.Getenv("AWS_TESTING_ACCOUNT_ID"))
		expectedInlinePolicy := `{"Version": "2012-10-17", "Statement": {"Effect": "Deny", "Action": "s3:*", "Resource": "arn:aws:s3:::random-test-bucket-us-west-1"}}`

		fetchIAMRole := &awsv1beta1.IAMRole{}
		c.EventuallyGet(helpers.Name("test"), fetchIAMRole, c.EventuallyStatus(awsv1beta1.StatusReady))

		Expect(roleExists(expectedRoleName)).ToNot(HaveOccurred())
		Expect(roleHasValidTag(expectedRoleName)).To(BeTrue())
		Expect(getRolePolicyNames(expectedRoleName)).To(HaveLen(2))
		Expect(getRolePolicyDocument(expectedRoleName, "test_tmpl")).To(MatchJSON(expectedInlinePolicy))
		Expect(getAssumeRolePolicyDocument(expectedRoleName)).To(MatchJSON(expectedAssumePolicy))

		Expect(fetchIAMRole.ObjectMeta.Finalizers).To(HaveLen(1))
		Expect(fetchIAMRole.ObjectMeta.DeletionTimestamp.IsZero()).To(BeTrue())
	})
})

func roleExists(roleName string) error {
	_, err := iamsvc.GetRole(&iam.GetRoleInput{RoleName: aws.String(iamRole.Spec.RoleName)})
	return err
}

func roleHasValidTag(roleName string) bool {
	listRoleTagsOutput, err := iamsvc.ListRoleTags(&iam.ListRoleTagsInput{RoleName: aws.String(iamRole.Spec.RoleName)})
	if err != nil {
		return false
	}

	var foundOperatorTag, foundKiamTag bool
	for _, TagSet := range listRoleTagsOutput.Tags {
		if aws.StringValue(TagSet.Key) == "ridecell-operator" && aws.StringValue(TagSet.Value) == "True" {
			foundOperatorTag = true
		}
		if aws.StringValue(TagSet.Key) == "iam:ResourceTag/Kiam" && aws.StringValue(TagSet.Value) == "true" {
			foundKiamTag = true
		}
	}

	if foundOperatorTag && foundKiamTag {
		return true
	}
	return false
}

func getRolePolicyNames(roleName string) []*string {
	getRolePoliciesOutput, err := iamsvc.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: aws.String(roleName)})
	if err != nil {
		return nil
	}
	return getRolePoliciesOutput.PolicyNames
}

func getRolePolicyDocument(roleName string, rolePolicyName string) string {
	getRolePolicy, err := iamsvc.GetRolePolicy(&iam.GetRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(rolePolicyName),
	})
	if err != nil {
		return ""
	}

	decoded, err := url.PathUnescape(aws.StringValue(getRolePolicy.PolicyDocument))
	if err != nil {
		return ""
	}
	return decoded
}

func getAssumeRolePolicyDocument(roleName string) string {
	fetchRole, err := iamsvc.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		return err.Error()
	}

	return aws.StringValue(fetchRole.Role.AssumeRolePolicyDocument)
}
