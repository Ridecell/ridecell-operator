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
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockEC2SGClient struct {
	ec2iface.EC2API
	securityGroupExists    bool
	hasValidIpRange        bool
	hasInvalidExtraRule    bool
	hasInvalidPortRule     bool
	hasValidTags           bool
	createdSG              bool
	authorizedDefaultRule  bool
	authorizedExtraRule    bool
	revokedExtraRule       bool
	revokedInvalidPortRule bool
	createdTag             bool
	deletedSecurityGroup   bool
}

type mockRDSSGClient struct {
	rdsiface.RDSAPI
}

var _ = Describe("rds security group Component", func() {
	os.Setenv("RDS_SG_RULES", "{\"1.2.3.4/32\":\"custom1\"}")
	comp := rdscomponents.NewDBSecurityGroup()
	var mockEC2 *mockEC2SGClient
	var mockRDS *mockRDSSGClient

	BeforeEach(func() {
		comp = rdscomponents.NewDBSecurityGroup()
		mockEC2 = &mockEC2SGClient{}
		mockRDS = &mockRDSSGClient{}
		comp.InjectAWSAPIs(mockEC2, mockRDS)
		instance.Spec.VPCID = "test"
		instance.ObjectMeta.Finalizers = []string{"rdsinstance.securitygroup.finalizer"}
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
		Expect(mockEC2.authorizedExtraRule).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeFalse())

		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.authorizedExtraRule).To(BeTrue())
		Expect(mockEC2.createdTag).To(BeTrue())
	})

	It("makes no changes", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIpRange = true
		mockEC2.hasValidTags = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedExtraRule).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeFalse())
	})

	It("has existing security group with no tag", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIpRange = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedExtraRule).To(BeFalse())
		Expect(mockEC2.createdTag).To(BeTrue())
	})

	It("has existing security group with bad ingress", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidTags = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.authorizedExtraRule).To(BeTrue())
		Expect(mockEC2.createdTag).To(BeFalse())
	})

	It("revokes invalid port sg rule", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIpRange = true
		mockEC2.hasInvalidPortRule = true
		mockEC2.hasValidTags = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.revokedInvalidPortRule).To(BeTrue())
	})

	It("revokes invalid sg rule", func() {
		mockEC2.securityGroupExists = true
		mockEC2.hasValidIpRange = true
		mockEC2.hasInvalidExtraRule = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockEC2.createdSG).To(BeFalse())
		Expect(mockEC2.revokedExtraRule).To(BeTrue())
	})

	// It("adds default sg rule when environment var not defined", func() {
	// 	os.Setenv("RDS_SG_RULES", "")
	// 	mockEC2.securityGroupExists = true
	// 	mockEC2.hasValidIpRange = false
	// 	Expect(comp).To(ReconcileContext(ctx))
	// 	Expect(mockEC2.createdSG).To(BeFalse())
	// 	Expect(mockEC2.authorizedDefaultRule).To(BeTrue())
	// })

	It("tests adding the finalizer", func() {
		instance.ObjectMeta.Finalizers = []string{}
		Expect(comp).To(ReconcileContext(ctx))

		fetchRDSInstance := &dbv1beta1.RDSInstance{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test", Namespace: "default"}, fetchRDSInstance)
		Expect(err).ToNot(HaveOccurred())

		Expect(fetchRDSInstance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.securitygroup.finalizer"))
	})

	It("test finalizer behavior during deletion", func() {
		os.Setenv("ENABLE_FINALIZERS", "true")
		mockEC2.securityGroupExists = true
		currentTime := metav1.Now()
		instance.ObjectMeta.SetDeletionTimestamp(&currentTime)

		Expect(comp).To(ReconcileContext(ctx))

		fetchRDSInstance := &dbv1beta1.RDSInstance{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test", Namespace: "default"}, fetchRDSInstance)
		Expect(err).ToNot(HaveOccurred())
		Expect(mockEC2.deletedSecurityGroup).To(BeTrue())
		Expect(fetchRDSInstance.ObjectMeta.Finalizers).To(HaveLen(0))
	})

})

// Mock aws functions below
func (m *mockEC2SGClient) DescribeSecurityGroups(input *ec2.DescribeSecurityGroupsInput) (*ec2.DescribeSecurityGroupsOutput, error) {
	if aws.StringValue(input.Filters[0].Values[0]) != "ridecell-operator-rds-test" {
		return nil, errors.New("mock_ec2: input security group name did not match expected value")
	}
	if m.securityGroupExists {
		securityGroup := &ec2.SecurityGroup{
			GroupId: aws.String("abcdf-1293238923"),
		}
		if m.hasValidIpRange {
			ipList := []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("1.2.3.4/32")}}
			if m.authorizedDefaultRule {
				ipList = []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")}}
			}
			if m.hasInvalidExtraRule {
				ipList = append(ipList, &ec2.IpRange{CidrIp: aws.String("1.0.0.0/0")})
			}
			securityGroup.IpPermissions = []*ec2.IpPermission{
				&ec2.IpPermission{
					FromPort: aws.Int64(int64(5432)),
					ToPort:   aws.Int64(int64(5432)),
					IpRanges: ipList,
				},
			}
			if m.hasInvalidPortRule {
				securityGroup.IpPermissions = append(securityGroup.IpPermissions, &ec2.IpPermission{
					FromPort: aws.Int64(int64(80)),
					ToPort:   aws.Int64(int64(80)),
					IpRanges: ipList,
				})
			}
		}
		if m.hasValidTags {
			securityGroup.Tags = []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Ridecell-Operator"),
					Value: aws.String("true"),
				},
				&ec2.Tag{
					Key:   aws.String("tenant"),
					Value: aws.String(instance.Name),
				},
			}
		}
		return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: []*ec2.SecurityGroup{securityGroup}}, nil
	}
	return &ec2.DescribeSecurityGroupsOutput{}, nil
}

func (m *mockEC2SGClient) CreateSecurityGroup(input *ec2.CreateSecurityGroupInput) (*ec2.CreateSecurityGroupOutput, error) {
	if aws.StringValue(input.GroupName) != "ridecell-operator-rds-test" {
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
	if aws.StringValue(input.IpPermissions[0].IpRanges[0].CidrIp) == "0.0.0.0/0" {
		m.authorizedDefaultRule = true
	} else {
		m.authorizedExtraRule = true
	}
	return nil, nil
}

func (m *mockEC2SGClient) RevokeSecurityGroupIngress(input *ec2.RevokeSecurityGroupIngressInput) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	if aws.StringValue(input.GroupId) != "abcdf-1293238923" {
		return nil, errors.New("mock_ec2: input security group id did not match expected value")
	}
	if aws.Int64Value(input.IpPermissions[0].FromPort) != int64(5432) {
		m.revokedInvalidPortRule = true
	} else {
		m.revokedExtraRule = true
	}
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

func (m *mockRDSSGClient) DescribeDBSubnetGroups(input *rds.DescribeDBSubnetGroupsInput) (*rds.DescribeDBSubnetGroupsOutput, error) {
	return &rds.DescribeDBSubnetGroupsOutput{
		DBSubnetGroups: []*rds.DBSubnetGroup{
			&rds.DBSubnetGroup{VpcId: aws.String("test")},
		},
	}, nil
}
