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
	"fmt"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
)

const rdsInstanceSecurityGroupFinalizer = "rdsinstance.securitygroup.finalizer"

type dbSecurityGroupComponent struct {
	ec2API ec2iface.EC2API
	rdsAPI rdsiface.RDSAPI
}

func NewDBSecurityGroup() *dbSecurityGroupComponent {
	sess := session.Must(session.NewSession())
	ec2Service := ec2.New(sess)
	rdsService := rds.New(sess)
	return &dbSecurityGroupComponent{
		ec2API: ec2Service,
		rdsAPI: rdsService,
	}
}

func (comp *dbSecurityGroupComponent) InjectAWSAPIs(ec2api ec2iface.EC2API, rdsapi rdsiface.RDSAPI) {
	comp.ec2API = ec2api
	comp.rdsAPI = rdsapi
}

func (_ *dbSecurityGroupComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *dbSecurityGroupComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *dbSecurityGroupComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)

	securityGroupName := fmt.Sprintf("ridecell-operator-rds-%s", instance.Name)

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(rdsInstanceSecurityGroupFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(rdsInstanceSecurityGroupFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(rdsInstanceSecurityGroupFinalizer, instance) {
			// If our database still exists we can't delete the security group
			if helpers.ContainsFinalizer(RDSInstanceDatabaseFinalizer, instance) {
				return components.Result{RequeueAfter: time.Minute * 1}, nil
			}
			if !instance.Spec.SkipFinalizers {
				result, err := comp.deleteDependencies(ctx)
				if err != nil {
					return result, err
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(rdsInstanceSecurityGroupFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to update instance while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer exit.
		return components.Result{}, nil
	}

	describeSecurityGroupsOutput, err := comp.ec2API.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(securityGroupName)},
			},
		},
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to describe security group")
	}

	if len(describeSecurityGroupsOutput.SecurityGroups) < 1 {
		vpcID, err := comp.getVPCID(ctx)
		if err != nil {
			return components.Result{}, err
		}
		_, err = comp.ec2API.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(securityGroupName),
			Description: aws.String(fmt.Sprintf("%s: Created by ridecell-operator", securityGroupName)),
			VpcId:       vpcID,
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
	var foundOperatorTag bool
	var foundTenantTag bool
	for _, tagSet := range securityGroup.Tags {
		if aws.StringValue(tagSet.Key) == "Ridecell-Operator" && aws.StringValue(tagSet.Value) == "true" {
			foundOperatorTag = true
		}
		if aws.StringValue(tagSet.Key) == "tenant" && aws.StringValue(tagSet.Value) == instance.Name {
			foundTenantTag = true
		}
	}

	if !foundOperatorTag || !foundTenantTag {
		_, err := comp.ec2API.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{securityGroup.GroupId},
			Tags: []*ec2.Tag{
				&ec2.Tag{
					Key:   aws.String("Ridecell-Operator"),
					Value: aws.String("true"),
				},
				&ec2.Tag{
					Key:   aws.String("tenant"),
					Value: aws.String(instance.Name),
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to tag security group")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RDSInstance)
		instance.Status.SecurityGroupID = aws.StringValue(securityGroup.GroupId)
		return nil
	}}, nil
}

func (comp *dbSecurityGroupComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)
	describeSecurityGroupsOutput, err := comp.ec2API.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(fmt.Sprintf("ridecell-operator-rds-%s", instance.Name))},
			},
		},
	})

	// This shouldn't happen but leaving it here for sanity
	if len(describeSecurityGroupsOutput.SecurityGroups) < 1 {
		// Our security group no longer exists
		return components.Result{}, nil
	}

	_, err = comp.ec2API.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: describeSecurityGroupsOutput.SecurityGroups[0].GroupId,
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to delete security group for finalizer")
	}
	// SecurityGroup in the process of being deleted
	return components.Result{}, nil
}

func (comp *dbSecurityGroupComponent) getVPCID(ctx *components.ComponentContext) (*string, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)
	describeDBSubnetGroups, err := comp.rdsAPI.DescribeDBSubnetGroups(&rds.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(instance.Spec.SubnetGroupName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "rds: failed to describe subnet group")
	}
	return describeDBSubnetGroups.DBSubnetGroups[0].VpcId, nil
}
