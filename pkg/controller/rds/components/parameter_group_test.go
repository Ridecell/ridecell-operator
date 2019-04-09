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
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"

	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
)

type mockRDSPGClient struct {
	rdsiface.RDSAPI

	parameterGroupExists    bool
	parameterGroupHasParams bool

	modifiedParameters bool

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
		Expect(mockRDS.modifiedParameters).To(BeTrue())

		Expect(parametersEquals(mockRDS.parameters, mockRDS.defaultParameters)).To(BeTrue())
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
	return &rds.CreateDBParameterGroupOutput{}, nil
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

func parametersEquals(listOne []*rds.Parameter, listTwo []*rds.Parameter) bool {
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
