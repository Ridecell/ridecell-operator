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

package components_test

import (
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	iamrolecomponents "github.com/Ridecell/ridecell-operator/pkg/controller/iamrole/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockIAMClient struct {
	iamiface.IAMAPI
	mockRoleExists      bool
	mockRoleHasTags     bool
	mockhasRolePolicies bool
	mockExtraRolePolicy bool
	mockRoleTagged      bool
	mockRoleCreated     bool

	deleteRole    bool
	finalizerTest bool

	expectedRoleName       string
	expectedPolicyDocument string
}

var _ = Describe("iam_role aws Component", func() {
	comp := iamrolecomponents.NewIAMRole()
	var mockIAM *mockIAMClient

	BeforeEach(func() {
		comp = iamrolecomponents.NewIAMRole()
		mockIAM = &mockIAMClient{}
		comp.InjectIAMAPI(mockIAM)
		// Finalizer is added here to skip the return in reconcile after adding finalizer
		instance.ObjectMeta.Finalizers = []string{"iamrole.finalizer"}
		// This needs to be valid json
		instance.Spec.AssumeRolePolicyDocument = "{}"

		instance.Spec.RoleName = "test-role"

		mockIAM.expectedRoleName = instance.Name
		mockIAM.expectedPolicyDocument = instance.Spec.AssumeRolePolicyDocument
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("runs basic reconcile with no existing role", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockIAM.mockRoleCreated).To(BeTrue())
	})

	It("has extra items attached to role", func() {
		mockIAM.mockRoleHasTags = true
		mockIAM.mockRoleExists = true
		mockIAM.mockExtraRolePolicy = true

		Expect(comp).To(ReconcileContext(ctx))
	})

	It("creates new role with policies", func() {
		instance.Spec.InlinePolicies = map[string]string{
			"test777": `{"Version": "2012-10-17", "Statement": {"Effect": "Allow", "Action": "s3:*", "Resource": "*"}}`,
		}

		Expect(comp).To(ReconcileContext(ctx))
	})

	It("creates new role with all the templates", func() {
		mockIAM.expectedRoleName = "test-role-us-test-1"
		mockIAM.expectedPolicyDocument = `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow","Principal": {"AWS": "arn:aws:iam:us-test-1:*"},"Action": "sts:AssumeRole"}]}`

		instance.Spec.AssumeRolePolicyDocument = `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow","Principal": {"AWS": "arn:aws:iam:{{ .Region }}:*"},"Action": "sts:AssumeRole"}]}`
		instance.Spec.InlinePolicies = map[string]string{
			"test777": `{"Version": "2012-10-17", "Statement": {"Effect": "Allow", "Action": "s3:*", "Resource": "arn::{{ .Region }}:*"}}`,
		}
		instance.Spec.RoleName = "test-role-{{ .Region }}"
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("errors on an invalid policy", func() {
		instance.Spec.InlinePolicies = map[string]string{
			"test": `{nope`,
		}

		_, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError("iam_role: role policy from spec test has invalid JSON: invalid character 'n' looking for beginning of object key string"))
	})

	It("makes sure component errors if role not properly tagged", func() {
		mockIAM.mockRoleExists = true

		_, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError("iam_role: existing role is not tagged with ridecell-operator: True, aborting"))
	})

	Describe("finalizer tests", func() {

		It("adds finalizer when there isn't one", func() {
			mockIAM.finalizerTest = true
			instance.ObjectMeta.Finalizers = []string{}

			Expect(comp).To(ReconcileContext(ctx))

			fetchIAMRole := &awsv1beta1.IAMRole{}
			err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "test-role", Namespace: "default"}, fetchIAMRole)
			Expect(err).ToNot(HaveOccurred())

			Expect(fetchIAMRole.ObjectMeta.Finalizers).To(Equal([]string{"iamrole.finalizer"}))
		})

		It("sets deletiontimestamp to non-zero with mock objects existing", func() {
			mockIAM.finalizerTest = true
			mockIAM.mockhasRolePolicies = true
			mockIAM.mockRoleExists = true
			currentTime := metav1.Now()
			instance.ObjectMeta.SetDeletionTimestamp(&currentTime)
			instance.Spec.InlinePolicies = map[string]string{
				"test777": `{"Version": "2012-10-17", "Statement": {"Effect": "Allow", "Action": "s3:*", "Resource": "*"}}`,
			}

			Expect(comp).To(ReconcileContext(ctx))

			fetchIAMRole := &awsv1beta1.IAMRole{}
			err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "test-role", Namespace: "default"}, fetchIAMRole)
			Expect(err).ToNot(HaveOccurred())

			Expect(mockIAM.deleteRole).To(BeTrue())
		})

		It("simulates role not existing during finalizer deletion", func() {
			currentTime := metav1.Now()
			mockIAM.finalizerTest = true
			instance.ObjectMeta.SetDeletionTimestamp(&currentTime)

			Expect(comp).To(ReconcileContext(ctx))

			fetchIAMRole := &awsv1beta1.IAMRole{}
			err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "test-role", Namespace: "default"}, fetchIAMRole)
			Expect(err).ToNot(HaveOccurred())

			Expect(fetchIAMRole.ObjectMeta.Finalizers).To(HaveLen(0))
		})
	})
})

// Mock aws functions below

func (m *mockIAMClient) GetRole(input *iam.GetRoleInput) (*iam.GetRoleOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.GetRoleOutput{}, errors.New("awsmock_getrole: given rolename does not match expected")
	}
	if m.mockRoleExists {
		return &iam.GetRoleOutput{Role: &iam.Role{RoleName: input.RoleName, AssumeRolePolicyDocument: aws.String(instance.Spec.AssumeRolePolicyDocument)}}, nil
	}
	return &iam.GetRoleOutput{}, awserr.New(iam.ErrCodeNoSuchEntityException, "awsmock_getrole: role does not exist", errors.New(""))
}

func (m *mockIAMClient) CreateRole(input *iam.CreateRoleInput) (*iam.CreateRoleOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.CreateRoleOutput{}, errors.New("awsmock_createrole: given rolename does not match expected")
	}
	if aws.StringValue(input.AssumeRolePolicyDocument) != m.expectedPolicyDocument {
		return &iam.CreateRoleOutput{}, errors.New("awsmock_createrole: given assume role policy document does not match spec")
	}
	m.mockRoleCreated = true
	m.mockRoleHasTags = true
	return &iam.CreateRoleOutput{Role: &iam.Role{RoleName: input.RoleName, AssumeRolePolicyDocument: input.AssumeRolePolicyDocument}}, nil
}

func (m *mockIAMClient) ListRolePolicies(input *iam.ListRolePoliciesInput) (*iam.ListRolePoliciesOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName || (!m.mockRoleExists && m.finalizerTest) {
		return &iam.ListRolePoliciesOutput{}, awserr.New(iam.ErrCodeNoSuchEntityException, "awsmock_listrolepolicies: given rolename does not match expected", errors.New(""))
	}
	if m.mockhasRolePolicies {
		inlinePoliciesPointers := []*string{}
		for k := range instance.Spec.InlinePolicies {
			inlinePoliciesPointers = append(inlinePoliciesPointers, aws.String(k))
		}
		return &iam.ListRolePoliciesOutput{PolicyNames: inlinePoliciesPointers}, nil
	}
	if m.mockExtraRolePolicy {
		inlinePoliciesPointers := []*string{}
		for k := range instance.Spec.InlinePolicies {
			inlinePoliciesPointers = append(inlinePoliciesPointers, aws.String(k))
		}
		inlinePoliciesPointers = append(inlinePoliciesPointers, aws.String("mock1"))
		return &iam.ListRolePoliciesOutput{PolicyNames: inlinePoliciesPointers}, nil
	}
	return &iam.ListRolePoliciesOutput{}, nil
}

func (m *mockIAMClient) GetRolePolicy(input *iam.GetRolePolicyInput) (*iam.GetRolePolicyOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.GetRolePolicyOutput{}, errors.New("awsmock_getrolepolicy: given rolename does not match expected")
	}
	if m.mockhasRolePolicies {
		inputPolicy := instance.Spec.InlinePolicies[aws.StringValue(input.PolicyName)]
		return &iam.GetRolePolicyOutput{PolicyName: input.PolicyName, PolicyDocument: aws.String(inputPolicy)}, nil
	}
	if m.mockExtraRolePolicy {
		inputPolicy, ok := instance.Spec.InlinePolicies[aws.StringValue(input.PolicyName)]
		if !ok {
			inputPolicy = `{"Version": "2012-10-17", "Statement": {"Effect": "Allow", "Action": ["s3:*"] "Resource": "*"}}`
		}
		return &iam.GetRolePolicyOutput{PolicyName: input.PolicyName, PolicyDocument: aws.String(inputPolicy)}, nil
	}
	return &iam.GetRolePolicyOutput{}, nil
}

func (m *mockIAMClient) PutRolePolicy(input *iam.PutRolePolicyInput) (*iam.PutRolePolicyOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.PutRolePolicyOutput{}, errors.New("awsmock_putrolepolicy: rolename did not match expected")
	}
	return &iam.PutRolePolicyOutput{}, nil
}

func (m *mockIAMClient) DeleteRolePolicy(input *iam.DeleteRolePolicyInput) (*iam.DeleteRolePolicyOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.DeleteRolePolicyOutput{}, errors.New("awsmock_deleterolepolicy: rolename did not match expected")
	}
	_, ok := instance.Spec.InlinePolicies[aws.StringValue(input.PolicyName)]
	if !ok || m.finalizerTest {
		return &iam.DeleteRolePolicyOutput{}, nil
	}
	return &iam.DeleteRolePolicyOutput{}, errors.New("awsmock_deleterolepolicy: policy shouldn't be getting deleted")
}

func (m *mockIAMClient) ListRoleTags(input *iam.ListRoleTagsInput) (*iam.ListRoleTagsOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.ListRoleTagsOutput{}, awserr.New(iam.ErrCodeNoSuchEntityException, "awsmock_listroletags: rolename did not match expected", errors.New(""))
	}

	if m.mockRoleHasTags {
		return &iam.ListRoleTagsOutput{
			Tags: []*iam.Tag{
				&iam.Tag{
					Key:   aws.String("ridecell-operator"),
					Value: aws.String("True"),
				},
				&iam.Tag{
					Key:   aws.String("Kiam"),
					Value: aws.String("true"),
				},
			},
		}, nil
	}

	return &iam.ListRoleTagsOutput{}, nil
}

func (m *mockIAMClient) TagRole(input *iam.TagRoleInput) (*iam.TagRoleOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.TagRoleOutput{}, awserr.New(iam.ErrCodeNoSuchEntityException, "awsmock_tagrole: rolename did not match expected", errors.New(""))
	}
	m.mockRoleTagged = true
	return &iam.TagRoleOutput{}, nil
}

func (m *mockIAMClient) DeleteRole(input *iam.DeleteRoleInput) (*iam.DeleteRoleOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName || (!m.mockRoleExists && m.finalizerTest) {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "awsmock_deleterole: rolename did not match expected", errors.New(""))
	}
	m.deleteRole = true
	return &iam.DeleteRoleOutput{}, nil
}

func (m *mockIAMClient) UpdateAssumeRolePolicy(input *iam.UpdateAssumeRolePolicyInput) (*iam.UpdateAssumeRolePolicyOutput, error) {
	if aws.StringValue(input.RoleName) != m.expectedRoleName {
		return &iam.UpdateAssumeRolePolicyOutput{}, errors.New("awsmock_updateassumerolepolicy: rolename did not match expected")
	}
	return &iam.UpdateAssumeRolePolicyOutput{}, nil
}
