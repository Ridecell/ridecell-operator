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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"

	escomponents "github.com/Ridecell/ridecell-operator/pkg/controller/elasticsearch/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

type mockRDSClient struct {
	rdsiface.RDSAPI
}

var _ = Describe("ElasticSearch Defaults Component", func() {
	os.Setenv("AWS_SUBNET_GROUP_NAME", "test-subnet")
	os.Setenv("AWS_REGION", "us-west-2")
	comp := escomponents.NewDefaults()
	var mockRDS *mockRDSClient

	BeforeEach(func() {
		os.Setenv("AWS_SUBNET_GROUP_NAME", "test-subnet")
		os.Setenv("AWS_REGION", "us-west-2")
		comp = escomponents.NewDefaults()
		mockRDS = &mockRDSClient{}
		comp.InjectAPI(mockRDS)
	})

	It("does nothing on a filled out object", func() {
		instance.Spec.DeploymentType = "Production"
		instance.Spec.InstanceType = "r4.large.elasticsearch"
		instance.Spec.NoOfInstances = 1
		instance.Spec.ElasticSearchVersion = "5.0"
		instance.Spec.StoragePerNode = 20

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.DeploymentType).To(Equal("Production"))
		Expect(instance.Spec.InstanceType).To(Equal("r4.large.elasticsearch"))
		Expect(instance.Spec.ElasticSearchVersion).To(Equal("5.0"))
		Expect(instance.Spec.StoragePerNode).To(Equal(int64(20)))
		Expect(instance.Spec.VPCID).To(Equal("vpc-1234567890"))
		Expect(instance.Spec.SubnetIds[0]).To(Equal("subnet-12345"))
	})

	It("sets defaults", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.DeploymentType).To(Equal("Development"))
		Expect(instance.Spec.InstanceType).To(Equal("r5.large.elasticsearch"))
		Expect(instance.Spec.ElasticSearchVersion).To(Equal("7.1"))
		Expect(instance.Spec.StoragePerNode).To(Equal(int64(30)))
		Expect(instance.Spec.VPCID).To(Equal("vpc-1234567890"))
		Expect(instance.Spec.SubnetIds[0]).To(Equal("subnet-12345"))
	})

})

func (m *mockRDSClient) DescribeDBSubnetGroups(input *rds.DescribeDBSubnetGroupsInput) (*rds.DescribeDBSubnetGroupsOutput, error) {
	if aws.StringValue(input.DBSubnetGroupName) != "test-subnet" {
		return nil, errors.New("awsmock_describedbsubnetgroup: db subnet group does not match spec")
	}
	dbSubnetGroup := &rds.DescribeDBSubnetGroupsOutput{
		DBSubnetGroups: []*rds.DBSubnetGroup{
			&rds.DBSubnetGroup{
				VpcId: aws.String("vpc-1234567890"),
				Subnets: []*rds.Subnet{
					&rds.Subnet{
						SubnetIdentifier: aws.String("subnet-12345"),
					},
				},
			},
		},
	}
	return dbSubnetGroup, nil
}
