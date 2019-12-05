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

package components

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type defaultsComponent struct {
	rdsAPI rdsiface.RDSAPI
	ec2API ec2iface.EC2API
}

func NewDefaults() *defaultsComponent {
	sess := session.Must(session.NewSession())
	rdsService := rds.New(sess)
	ec2Service := ec2.New(sess)
	return &defaultsComponent{rdsAPI: rdsService, ec2API: ec2Service}
}

func (comp *defaultsComponent) InjectAPI(rdsapi rdsiface.RDSAPI, ec2api ec2iface.EC2API) {
	comp.rdsAPI = rdsapi
	comp.ec2API = ec2api
}

func (_ *defaultsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *defaultsComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *defaultsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.ElasticSearch)

	// Populate the VPC, Subnet and Sucurity group
	describeDBSubnetGroupOutput, err := comp.rdsAPI.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: aws.String(os.Getenv("AWS_SUBNET_GROUP_NAME"))})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to describe subnet group")
	}
	// Fill in defaults.
	instance.Spec.VPCID = aws.StringValue(describeDBSubnetGroupOutput.DBSubnetGroups[0].VpcId)

	subnetIdList := []string{}
	for _, subnet := range describeDBSubnetGroupOutput.DBSubnetGroups[0].Subnets {
		subnetIdList = append(subnetIdList, aws.StringValue(subnet.SubnetIdentifier))
	}
	instance.Spec.SubnetIds = subnetIdList

	if instance.Spec.SecurityGroupId == "" {
		// Get Security group Id
		describeSecurityGroupsOutput, err := comp.ec2API.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{&ec2.Filter{
				Name:   aws.String("tag:Name"),
				Values: []*string{aws.String(os.Getenv("AWS_SUBNET_GROUP_NAME"))},
			},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to describe security group")
		}
		instance.Spec.SecurityGroupId = aws.StringValue(describeSecurityGroupsOutput.SecurityGroups[0].GroupId)
	}

	if instance.Spec.DeploymentType == "" {
		instance.Spec.DeploymentType = "Development"
	}

	if instance.Spec.InstanceType == "" {
		instance.Spec.InstanceType = "r5.large.elasticsearch"
	}

	if instance.Spec.NoOfInstances == 0 {
		instance.Spec.NoOfInstances = 1
		if instance.Spec.DeploymentType == "Production" {
			// No of instances must be multiple of no. of subnet group available
			instance.Spec.NoOfInstances = int64(len(subnetIdList))
		}
	}

	if instance.Spec.ElasticSearchVersion == "" {
		instance.Spec.ElasticSearchVersion = "7.1"
	}

	if instance.Spec.StoragePerNode == 0 {
		instance.Spec.StoragePerNode = 10 // 10 GB
	}

	if instance.Spec.SnapshotTime == "" {
		instance.Spec.SnapshotTime = "00:00 UTC"
	}

	return components.Result{}, nil
}
