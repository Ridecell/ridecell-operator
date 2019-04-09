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
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
)

type dbSecurityGroupComponent struct {
	ec2API ec2iface.EC2API
}

func NewDBSecurityGroup() *dbSecurityGroupComponent {
	sess := session.Must(session.NewSession())
	ec2Service := ec2.New(sess)
	return &dbSecurityGroupComponent{ec2API: ec2Service}
}

func (comp *dbSecurityGroupComponent) InjectEC2API(ec2api ec2iface.EC2API) {
	comp.ec2API = ec2api
}

func (_ *dbSecurityGroupComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *dbSecurityGroupComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *dbSecurityGroupComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)

	if instance.Spec.VPCID == "" {
		return components.Result{}, errors.New("rds: vpc_id environment variable not set")
	}

	describeSecurityGroupsOutput, err := comp.ec2API.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(instance.Name)},
			},
		},
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to describe security group")
	}

	if len(describeSecurityGroupsOutput.SecurityGroups) < 1 {
		_, err = comp.ec2API.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(instance.Name),
			Description: aws.String("Created by ridecell-operator"),
			VpcId:       aws.String(instance.Spec.VPCID),
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to create security group")
		}
		return components.Result{Requeue: true}, nil
	}
	securityGroup := describeSecurityGroupsOutput.SecurityGroups[0]

	var hasIngressRule bool
	for _, ipPermission := range securityGroup.IpPermissions {
		if aws.Int64Value(ipPermission.FromPort) != int64(5432) || aws.Int64Value(ipPermission.ToPort) != int64(5432) {
			continue
		}
		for _, ipRange := range ipPermission.IpRanges {
			if aws.StringValue(ipRange.CidrIp) == "0.0.0.0/0" {
				hasIngressRule = true
				break
			}
		}
	}

	if !hasIngressRule {
		_, err := comp.ec2API.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			CidrIp:     aws.String("0.0.0.0/0"),
			FromPort:   aws.Int64(int64(5432)),
			ToPort:     aws.Int64(int64(5432)),
			GroupId:    securityGroup.GroupId,
			IpProtocol: aws.String("tcp"),
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to authorize security group ingress")
		}
	}

	// welcome to fun mit tags
	var hasTag bool
	for _, tagSet := range securityGroup.Tags {
		if aws.StringValue(tagSet.Key) == "Ridecell-Operator" && aws.StringValue(tagSet.Value) == "true" {
			hasTag = true
		}
	}

	if !hasTag {
		_, err := comp.ec2API.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{securityGroup.GroupId},
			Tags: []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Ridecell-Operator"),
					Value: aws.String("true"),
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to tag security group")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RDSInstance)
		instance.Status.SecurityGroupStatus = dbv1beta1.StatusReady
		instance.Status.SecurityGroupID = aws.StringValue(securityGroup.GroupId)
		return nil
	}}, nil

}
