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
	"os"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
)

const elasticSearchSecurityGroupFinalizer = "elasticsearch.securitygroup.finalizer"

type esSecurityGroupComponent struct {
	ec2API ec2iface.EC2API
}

func NewESSecurityGroup() *esSecurityGroupComponent {
	sess := session.Must(session.NewSession())
	ec2Service := ec2.New(sess)
	return &esSecurityGroupComponent{ec2API: ec2Service}
}

func (comp *esSecurityGroupComponent) InjectAWSAPIs(ec2api ec2iface.EC2API) {
	comp.ec2API = ec2api
}

func (_ *esSecurityGroupComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *esSecurityGroupComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *esSecurityGroupComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.ElasticSearch)
	securityGroupName := fmt.Sprintf("ridecell-operator-es-%s", instance.Name)

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(elasticSearchSecurityGroupFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(elasticSearchSecurityGroupFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "elasticsearch: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(elasticSearchSecurityGroupFinalizer, instance) {
			// If our database still exists we can't delete the security group
			if helpers.ContainsFinalizer(elasticsearchFinalizer, instance) {
				return components.Result{RequeueAfter: time.Second * 5}, nil
			}
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" && os.Getenv("ENABLE_FINALIZERS") == "true" {
				result, err := comp.deleteDependencies(ctx)
				if err != nil {
					return result, err
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(elasticSearchSecurityGroupFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "elasticsearch: failed to update instance while removing finalizer")
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
		return components.Result{}, errors.Wrap(err, "elasticsearch: failed to describe security group")
	}

	if len(describeSecurityGroupsOutput.SecurityGroups) < 1 {
		// Create Security group
		_, err = comp.ec2API.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
			GroupName:   aws.String(securityGroupName),
			Description: aws.String(fmt.Sprintf("%s: Created by ridecell-operator", securityGroupName)),
			VpcId:       aws.String(instance.Spec.VPCID),
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "elasticsearch: failed to create security group")
		}
		return components.Result{Requeue: true}, nil
	}

	// Set the securityGroup field of instance spec
	instance.Spec.SecurityGroupId = aws.StringValue(describeSecurityGroupsOutput.SecurityGroups[0].GroupId)
	securityGroup := describeSecurityGroupsOutput.SecurityGroups[0]
	sgOutput, err := comp.ec2API.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{&ec2.Filter{
			Name:   aws.String("tag:Name"),
			Values: []*string{aws.String(fmt.Sprintf("nodes.%s", os.Getenv("AWS_SUBNET_GROUP_NAME")))},
		},
		},
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to describe node's security group")
	}
	nodeSecurityGroupId := aws.StringValue(sgOutput.SecurityGroups[0].GroupId)

	var hasIngressRule bool
	for _, ipPermission := range securityGroup.IpPermissions {
		if aws.StringValue(ipPermission.IpProtocol) == "-1" {
			for _, pair := range ipPermission.UserIdGroupPairs {
				if aws.StringValue(pair.GroupId) == nodeSecurityGroupId {
					hasIngressRule = true
					break
				}
			}
		}
	}

	if !hasIngressRule {
		_, err := comp.ec2API.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: securityGroup.GroupId,
			IpPermissions: []*ec2.IpPermission{
				{
					IpProtocol: aws.String("-1"),
					UserIdGroupPairs: []*ec2.UserIdGroupPair{
						{
							Description: aws.String("Allows ES Domain access inside VPC"),
							GroupId:     aws.String(nodeSecurityGroupId),
						},
					},
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "elasticsearch: failed to authorize security group ingress")
		}
	}

	// welcome to fun mit tags
	var foundOperatorTag bool
	for _, tagSet := range securityGroup.Tags {
		if aws.StringValue(tagSet.Key) == "Ridecell-Operator" && aws.StringValue(tagSet.Value) == "true" {
			foundOperatorTag = true
		}
	}

	if !foundOperatorTag {
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
			return components.Result{}, errors.Wrap(err, "elasticsearch: failed to tag security group")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*awsv1beta1.ElasticSearch)
		instance.Status.Status = "Processing"
		instance.Status.Message = "Security Group created"
		return nil
	}}, nil
}

func (comp *esSecurityGroupComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.ElasticSearch)
	describeSecurityGroupsOutput, _ := comp.ec2API.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(fmt.Sprintf("ridecell-operator-es-%s", instance.Name))},
			},
		},
	})

	// This shouldn't happen but leaving it here for sanity
	if len(describeSecurityGroupsOutput.SecurityGroups) < 1 {
		// Our security group no longer exists
		return components.Result{}, nil
	}

	_, err := comp.ec2API.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: describeSecurityGroupsOutput.SecurityGroups[0].GroupId,
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "elasticsearch: failed to delete security group for finalizer")
	}
	// SecurityGroup in the process of being deleted
	return components.Result{}, nil
}
