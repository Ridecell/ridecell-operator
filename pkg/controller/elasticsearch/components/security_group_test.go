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
	"os"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	escomponents "github.com/Ridecell/ridecell-operator/pkg/controller/elasticsearch/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockEC2SGClient struct {
	ec2iface.EC2API
	securityGroupExists  bool
	hasValidIngressRule  bool
	hasValidTags         bool
	createdSG            bool
	authorizedSG         bool
	createdTag           bool
	deletedSecurityGroup bool
}

var _ = Describe("elasticsearch security group Component", func() {
	os.Setenv("AWS_SUBNET_GROUP_NAME", "test-subnet")
	comp := escomponents.NewESSecurityGroup()
	var mockEC2 *mockEC2SGClient

	BeforeEach(func() {
		comp = escomponents.NewESSecurityGroup()
		mockEC2 = &mockEC2SGClient{}
		comp.InjectAWSAPIs(mockEC2)
		instance.Spec.VPCID = "test"
		instance.ObjectMeta.Finalizers = []string{"elasticsearch.securitygroup.finalizer"}
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
		Expect(instance.Spec.SecurityGroupId).To(Equal("abcdf-1293238923"))

	})

	It("makes no changes", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIngressRule = true
		mockEC2.hasValidTags = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedSG).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeFalse())
	})

	It("has existing security group with no tag", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIngressRule = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedSG).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeTrue())
	})

	It("tests adding the finalizer", func() {
		instance.ObjectMeta.Finalizers = []string{}
		Expect(comp).To(ReconcileContext(ctx))

		fetchESInstance := &awsv1beta1.ElasticSearch{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test-domain", Namespace: "default"}, fetchESInstance)
		Expect(err).ToNot(HaveOccurred())

		Expect(fetchESInstance.ObjectMeta.Finalizers[0]).To(Equal("elasticsearch.securitygroup.finalizer"))
	})

	It("test finalizer behavior during deletion", func() {
		os.Setenv("ENABLE_FINALIZERS", "true")
		mockEC2.securityGroupExists = true
		currentTime := metav1.Now()
		instance.ObjectMeta.SetDeletionTimestamp(&currentTime)

		Expect(comp).To(ReconcileContext(ctx))

		fetchESInstance := &awsv1beta1.ElasticSearch{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test-domain", Namespace: "default"}, fetchESInstance)
		Expect(err).ToNot(HaveOccurred())
		Expect(mockEC2.deletedSecurityGroup).To(BeTrue())
		Expect(fetchESInstance.ObjectMeta.Finalizers).To(HaveLen(0))
	})

})

// Mock aws functions below
func (m *mockEC2SGClient) DescribeSecurityGroups(input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
	if aws.StringValue(input.Filters[0].Values[0]) == "nodes.test-subnet" {
		return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []*ec2.SecurityGroup{&ec2.SecurityGroup{GroupId: aws.String("sg-1234567890")}}}, nil
	}
	if aws.StringValue(input.Filters[0].Values[0]) != "ridecell-operator-es-test-domain" {
		return nil, errors.New("mock_ec2: input security group name did not match expected value")
	}
	if m.securityGroupExists {
		securityGroup := &ec2.SecurityGroup{
			GroupId: aws.String("abcdf-1293238923"),
		}
		if m.hasValidIngressRule {
			securityGroup.IpPermissions = []*ec2.IpPermission{
				&ec2.IpPermission{
					IpProtocol: aws.String("-1"),
					UserIdGroupPairs: []*ec2.UserIdGroupPair{
						{
							Description: aws.String("Allows ES Domain access inside VPC"),
							GroupId:     aws.String("sg-1234567890"),
						},
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
	if aws.StringValue(input.GroupName) != "ridecell-operator-es-test-domain" {
		return nil, errors.New("mock_ec2: input security group name did not match expected value")
	}
	if aws.StringValue(input.VpcId) != "test" {
		return nil, errors.New("mock_ec2: input security group vpc id did not match expected value")
	}
	m.createdSG = true
	return &ec2.CreateSecurityGroupOutput{GroupId: aws.String("abcdf-1293238923")}, nil
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

func (m *mockEC2SGClient) DeleteSecurityGroup(input *ec2.DeleteSecurityGroupInput) (*ec2.DeleteSecurityGroupOutput, error) {
	m.deletedSecurityGroup = true
	return &ec2.DeleteSecurityGroupOutput{}, nil
}
