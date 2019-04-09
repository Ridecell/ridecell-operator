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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"

	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
)

type mockEC2SGClient struct {
	ec2iface.EC2API
	securityGroupExists bool
	hasValidIpRange     bool
	hasValidTags        bool

	createdSG    bool
	authorizedSG bool
	createdTag   bool
}

var _ = Describe("rds security group Component", func() {
	comp := rdscomponents.NewDBSecurityGroup()
	var mockEC2 *mockEC2SGClient

	BeforeEach(func() {
		comp = rdscomponents.NewDBSecurityGroup()
		mockEC2 = &mockEC2SGClient{}
		comp.InjectEC2API(mockEC2)
		instance.Spec.VPCID = "test"
	})

	Describe("isReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("runs through sg group creation from scratch", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeTrue())
		mockEC2.securityGroupExists = true
		Expect(mockEC2.authorizedSG).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeFalse())

		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.authorizedSG).To(BeTrue())
		Expect(mockEC2.createdTag).To(BeTrue())
	})

	It("makes no changes", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIpRange = true
		mockEC2.hasValidTags = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedSG).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeFalse())
	})

	It("has existing security group with no tag", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIpRange = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedSG).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeTrue())
	})

	It("has existing security group with bad ingress", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidTags = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedSG).To(BeTrue())
		Expect(mockEC2.createdTag).To(BeFalse())
	})

})

// Mock aws functions below
func (m *mockEC2SGClient) DescribeSecurityGroups(input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
	if aws.StringValue(input.Filters[0].Values[0]) != instance.Name {
		return nil, errors.New("mock_ec2: input security group name did not match expected value")
	}
	if m.securityGroupExists {
		securityGroup := &ec2.SecurityGroup{
			GroupId: aws.String("abcdf-1293238923"),
		}
		if m.hasValidIpRange {
			securityGroup.IpPermissions = []*ec2.IpPermission{
				&ec2.IpPermission{
					FromPort: aws.Int64(int64(5432)),
					ToPort:   aws.Int64(int64(5432)),
					IpRanges: []*ec2.IpRange{
						&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")},
					},
				},
			}
		}
		if m.hasValidTags {
			securityGroup.Tags = []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Ridecell-Operator"),
					Value: aws.String("true"),
				},
			}
		}
		return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []*ec2.SecurityGroup{securityGroup}}, nil
	}
	return &ec2.DescribeSecurityGroupsOutput{}, nil
}

func (m *mockEC2SGClient) CreateSecurityGroup(input *ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error) {
	if aws.StringValue(input.GroupName) != instance.Name {
		return nil, errors.New("mock_ec2: input security group name did not match expected value")
	}
	if aws.StringValue(input.VpcId) != "test" {
		return nil, errors.New("mock_ec2: input security group vpc id did not match expected value")
	}
	m.createdSG = true
	return nil, nil
}

func (m *mockEC2SGClient) AuthorizeSecurityGroupIngress(input *ec2.AuthorizeSecurityGroupIngressInput) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	if aws.StringValue(input.GroupId) != "abcdf-1293238923" {
		return nil, errors.New("mock_ec2: input security group id did not match expected value")
	}
	m.authorizedSG = true
	return nil, nil
}

func (m *mockEC2SGClient) CreateTags(input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	if aws.StringValue(input.Resources[0]) != "abcdf-1293238923" {
		return nil, errors.New("mock_ec2: resource id did not match expected value")
	}
	m.createdTag = true
	return nil, nil
}
