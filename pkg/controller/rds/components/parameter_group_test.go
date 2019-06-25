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
	"context"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockRDSPGClient struct {
	rdsiface.RDSAPI

	parameterGroupExists    bool
	parameterGroupHasParams bool
	modifiedParameters      bool
	defaultedParameters     bool
	deletedParameterGroup   bool
	hasTags                 bool
	addedTags               bool

	parameters        []*rds.Parameter
	defaultParameters []*rds.Parameter
}

var _ = Describe("rds parameter group Component", func() {
	comp := rdscomponents.NewDBParameterGroup()
	var mockRDS *mockRDSPGClient

	BeforeEach(func() {
		comp = rdscomponents.NewDBParameterGroup()
		mockRDS = &mockRDSPGClient{}
		comp.InjectRDSAPI(mockRDS)
		instance.Spec.Engine = "postgres"
		instance.Spec.EngineVersion = "11"
		instance.ObjectMeta.Finalizers = []string{"rdsinstance.parametergroup.finalizer"}

		mockRDS.defaultParameters = []*rds.Parameter{
			&rds.Parameter{
				ParameterName:  aws.String("test0"),
				ParameterValue: aws.String("test0"),
			},
			&rds.Parameter{
				ParameterName:  aws.String("test1"),
				ParameterValue: aws.String("true"),
			},
		}
		// pointer annoyances
		for _, defaultParam := range mockRDS.defaultParameters {
			nameString := aws.StringValue(defaultParam.ParameterName)
			valueString := aws.StringValue(defaultParam.ParameterValue)
			newParameter := &rds.Parameter{
				ParameterName:  aws.String(nameString),
				ParameterValue: aws.String(valueString),
			}
			mockRDS.parameters = append(mockRDS.parameters, newParameter)
		}
	})

	Describe("isReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("runs basic reconcile", func() {
		// Run again for the actual component
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("makes no change", func() {
		mockRDS.parameterGroupExists = true
		mockRDS.parameterGroupHasParams = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockRDS.modifiedParameters).To(BeFalse())
	})

	It("modifies group with missing parameters", func() {
		mockRDS.parameterGroupExists = true
		instance.Spec.Parameters = map[string]string{
			"test0": "newvalue",
		}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockRDS.modifiedParameters).To(BeTrue())
		Expect(mockRDS.parameters).To(HaveLen(2))

		for _, parameter := range mockRDS.parameters {
			if aws.StringValue(parameter.ParameterName) == "test0" {
				Expect(aws.StringValue(parameter.ParameterValue)).To(Equal("newvalue"))
			}
		}
	})

	It("resets a parameter to its default", func() {
		mockRDS.parameterGroupExists = true
		mockRDS.parameterGroupHasParams = true
		instance.Spec.Parameters = map[string]string{
			"test1": "newvalue",
		}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockRDS.modifiedParameters).To(BeFalse())

		instance.Spec.Parameters = map[string]string{}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockRDS.defaultedParameters).To(BeTrue())

		Expect(parametersEquals(mockRDS.parameters, mockRDS.defaultParameters)).To(BeTrue())
	})

	It("tests adding the finalizer", func() {
		instance.ObjectMeta.Finalizers = []string{}
		Expect(comp).To(ReconcileContext(ctx))

		fetchDBInstance := &dbv1beta1.RDSInstance{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test", Namespace: "default"}, fetchDBInstance)
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchDBInstance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.parametergroup.finalizer"))
	})

	It("tests finalizer behavior during deletion", func() {
		mockRDS.parameterGroupExists = true
		currentTime := metav1.Now()
		instance.ObjectMeta.SetDeletionTimestamp(&currentTime)

		Expect(comp).To(ReconcileContext(ctx))
		fetchDBInstance := &dbv1beta1.RDSInstance{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test", Namespace: "default"}, fetchDBInstance)
		Expect(err).ToNot(HaveOccurred())
		//Expect(mockRDS.deletedParameterGroup).To(BeTrue())
		Expect(fetchDBInstance.ObjectMeta.Finalizers).To(HaveLen(0))
	})
})

// Mock aws functions below
func (m *mockRDSPGClient) DescribeDBParameterGroups(input *rds.DescribeDBParameterGroupsInput) (*rds.DescribeDBParameterGroupsOutput, error) {
	if aws.StringValue(input.DBParameterGroupName) != instance.Name {
		return nil, errors.New("mock_rds: input parameter group name did not match expected value")
	}
	var parameterGroups []*rds.DBParameterGroup
	if m.parameterGroupExists {
		parameterGroups = []*rds.DBParameterGroup{
			&rds.DBParameterGroup{
				DBParameterGroupName: input.DBParameterGroupName,
				DBParameterGroupArn:  aws.String("arn"),
			},
		}
		return &rds.DescribeDBParameterGroupsOutput{DBParameterGroups: parameterGroups}, nil
	}
	return nil, awserr.New(rds.ErrCodeDBParameterGroupNotFoundFault, "", nil)
}

func (m *mockRDSPGClient) CreateDBParameterGroup(input *rds.CreateDBParameterGroupInput) (*rds.CreateDBParameterGroupOutput, error) {
	if aws.StringValue(input.DBParameterGroupName) != instance.Name {
		return nil, errors.New("mock_rds: input parameter group name did not match expected value")
	}
	if aws.StringValue(input.DBParameterGroupFamily) != "postgres11" {
		return nil, errors.New("mock_rds: input parameter group family did not match expected default")
	}
	return &rds.CreateDBParameterGroupOutput{
		DBParameterGroup: &rds.DBParameterGroup{
			DBParameterGroupArn: aws.String("arn"),
		},
	}, nil
}

func (m *mockRDSPGClient) DescribeDBParametersPages(input *rds.DescribeDBParametersInput, fn func(*rds.DescribeDBParametersOutput, bool) bool) error {
	if aws.StringValue(input.DBParameterGroupName) == "default.postgres11" {
		fn(&rds.DescribeDBParametersOutput{Parameters: m.defaultParameters}, false)
		return nil
	}
	if aws.StringValue(input.DBParameterGroupName) == "test" {
		if m.parameterGroupHasParams {
			for k, v := range instance.Spec.Parameters {
				for _, parameter := range m.parameters {
					if aws.StringValue(parameter.ParameterName) == k {
						parameter.ParameterValue = aws.String(v)
					}
				}
			}
		}
		fn(&rds.DescribeDBParametersOutput{Parameters: m.parameters}, false)
		return nil
	}
	return errors.New("mock_rds: unknown parameter group was queried")
}

// Why in the world does this single function differ from the rest of the sdk?????
func (m *mockRDSPGClient) ModifyDBParameterGroup(input *rds.ModifyDBParameterGroupInput) (*rds.DBParameterGroupNameMessage, error) {
	if aws.StringValue(input.DBParameterGroupName) != instance.Name {
		return nil, errors.New("mock_rds: input parameter group name did not match expected value")
	}

	for _, inputParameter := range input.Parameters {
		for _, parameter := range m.parameters {
			if aws.StringValue(inputParameter.ParameterName) == aws.StringValue(parameter.ParameterName) {
				valueString := aws.StringValue(inputParameter.ParameterValue)
				parameter.ParameterValue = aws.String(valueString)
			}
		}
	}

	m.modifiedParameters = true
	return nil, nil
}

func (m *mockRDSPGClient) ResetDBParameterGroup(input *rds.ResetDBParameterGroupInput) (*rds.DBParameterGroupNameMessage, error) {
	if aws.StringValue(input.DBParameterGroupName) != instance.Name {
		return nil, errors.New("mock_rds: input parameter group name did not match expected value")
	}
	if len(input.Parameters) > 20 {
		return nil, errors.New("mock_rds: more than 20 parameters given")
	}
	if aws.BoolValue(input.ResetAllParameters) {
		return nil, errors.New("mock_rds: resetallparameters should never be true")
	}

	// i hate everything about this
	for _, inputParameter := range input.Parameters {
		for _, parameter := range m.parameters {
			if aws.StringValue(parameter.ParameterName) == aws.StringValue(inputParameter.ParameterName) {
				for _, defaultParameter := range m.defaultParameters {
					if aws.StringValue(defaultParameter.ParameterName) == aws.StringValue(inputParameter.ParameterName) {
						defaultParameterValue := aws.StringValue(defaultParameter.ParameterValue)
						parameter.ParameterValue = aws.String(defaultParameterValue)
						break
					}
				}
				break
			}
		}
	}
	m.defaultedParameters = true
	return &rds.DBParameterGroupNameMessage{}, nil
}

func (m *mockRDSPGClient) DeleteDBParameterGroup(input *rds.DeleteDBParameterGroupInput) (*rds.DeleteDBParameterGroupOutput, error) {
	if aws.StringValue(input.DBParameterGroupName) != instance.Name {
		return nil, errors.New("mock_rds: input parameter group name did not match expected value")
	}
	m.deletedParameterGroup = true
	return &rds.DeleteDBParameterGroupOutput{}, nil
}

func (m *mockRDSPGClient) ListTagsForResource(input *rds.ListTagsForResourceInput) (*rds.ListTagsForResourceOutput, error) {
	if m.hasTags {
		tags := []*rds.Tag{
			&rds.Tag{
				Key:   aws.String("Ridecell-Operator"),
				Value: aws.String("true"),
			},
			&rds.Tag{
				Key:   aws.String("tenant"),
				Value: aws.String("test"),
			},
		}
		return &rds.ListTagsForResourceOutput{TagList: tags}, nil
	}
	return &rds.ListTagsForResourceOutput{}, nil
}

func (m *mockRDSPGClient) AddTagsToResource(input *rds.AddTagsToResourceInput) (*rds.AddTagsToResourceOutput, error) {
	m.addedTags = true
	return &rds.AddTagsToResourceOutput{}, nil
}

func parametersEquals(listOne []*rds.Parameter, listTwo []*rds.Parameter) bool {
	if len(listOne) != len(listTwo) {
		return false
	}
	for _, i := range listOne {
		var foundName bool
		var valueMatched bool
		for _, j := range listTwo {
			if aws.StringValue(i.ParameterName) == aws.StringValue(j.ParameterName) {
				foundName = true
				if aws.StringValue(j.ParameterValue) == aws.StringValue(j.ParameterValue) {
					valueMatched = true
				}
				break
			}
		}
		if !foundName || !valueMatched {
			return false
		}
	}
	return true
}
