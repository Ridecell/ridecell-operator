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
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
)

const rdsInstanceParameterGroupFinalizer = "rdsinstance.parametergroup.finalizer"

type dbParameterGroupComponent struct {
	rdsAPI rdsiface.RDSAPI
}

func NewDBParameterGroup() *dbParameterGroupComponent {
	sess := session.Must(session.NewSession())
	rdsService := rds.New(sess)
	return &dbParameterGroupComponent{rdsAPI: rdsService}
}

func (comp *dbParameterGroupComponent) InjectRDSAPI(rdsapi rdsiface.RDSAPI) {
	comp.rdsAPI = rdsapi
}

func (_ *dbParameterGroupComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *dbParameterGroupComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *dbParameterGroupComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(rdsInstanceParameterGroupFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(rdsInstanceParameterGroupFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrap(err, "rds: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(rdsInstanceParameterGroupFinalizer, instance) {
			// If our database still exists we can't delete the security group
			if helpers.ContainsFinalizer(RDSInstanceDatabaseFinalizer, instance) {
				return components.Result{RequeueAfter: time.Minute * 1}, nil
			}
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" && os.Getenv("ENABLE_FINALIZERS") == "true" {
				result, err := comp.deleteDependencies(ctx)
				if err != nil {
					return result, err
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(rdsInstanceParameterGroupFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrap(err, "rds: failed to update instance while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	var parameterGroup *rds.DBParameterGroup
	describeDBParameterGroupsOutput, err := comp.rdsAPI.DescribeDBParameterGroups(&rds.DescribeDBParameterGroupsInput{
		DBParameterGroupName: aws.String(instance.Name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == rds.ErrCodeDBParameterGroupNotFoundFault {
			createDBParameterGroupOutput, err := comp.rdsAPI.CreateDBParameterGroup(&rds.CreateDBParameterGroupInput{
				DBParameterGroupName:   aws.String(instance.Name),
				DBParameterGroupFamily: aws.String(fmt.Sprintf("%s%s", instance.Spec.Engine, instance.Spec.EngineVersion)),
				Description:            aws.String("Created by ridecell-operator"),
				Tags: []*rds.Tag{
					&rds.Tag{
						Key:   aws.String("Ridecell-Operator"),
						Value: aws.String("true"),
					},
					&rds.Tag{
						Key:   aws.String("tenant"),
						Value: aws.String(instance.Name),
					},
				},
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to create parameter group")
			}
			parameterGroup = createDBParameterGroupOutput.DBParameterGroup
		} else {
			return components.Result{}, errors.Wrapf(aerr, "rds: failed to describe parameter group")
		}
	} else {
		parameterGroup = describeDBParameterGroupsOutput.DBParameterGroups[0]
	}

	// handle tagging
	listTagsForResourceOutput, err := comp.rdsAPI.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: parameterGroup.DBParameterGroupArn,
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to list parameter group tags")
	}

	var foundOperatorTag bool
	var foundTenantTag bool
	for _, tag := range listTagsForResourceOutput.TagList {
		if aws.StringValue(tag.Key) == "Ridecell-Operator" && aws.StringValue(tag.Value) == "true" {
			foundOperatorTag = true
		}
		if aws.StringValue(tag.Key) == "tenant" && aws.StringValue(tag.Value) == instance.Name {
			foundTenantTag = true
		}
	}

	var tagsToAdd []*rds.Tag
	if !foundOperatorTag {
		tagsToAdd = append(tagsToAdd, &rds.Tag{Key: aws.String("Ridecell-Operator"), Value: aws.String("true")})
	}
	if !foundTenantTag {
		tagsToAdd = append(tagsToAdd, &rds.Tag{Key: aws.String("tentant"), Value: aws.String(instance.Name)})
	}
	if len(tagsToAdd) > 0 {
		_, err = comp.rdsAPI.AddTagsToResource(&rds.AddTagsToResourceInput{
			ResourceName: parameterGroup.DBParameterGroupArn,
			Tags:         tagsToAdd,
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to add tags to parameter group")
		}
	}

	// Get default parameter group values
	var defaultDBParams []*rds.Parameter
	err = comp.rdsAPI.DescribeDBParametersPages(&rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(fmt.Sprintf("default.%s%s", instance.Spec.Engine, instance.Spec.EngineVersion)),
	}, func(page *rds.DescribeDBParametersOutput, lastPage bool) bool {
		defaultDBParams = append(defaultDBParams, page.Parameters...)
		// if items returned < default MaxItems
		return !(len(page.Parameters) < 100)
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "rds: failed to describe db parameters")
	}

	// Get current parameter group values
	var dbParams []*rds.Parameter
	err = comp.rdsAPI.DescribeDBParametersPages(&rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(instance.Name),
	}, func(page *rds.DescribeDBParametersOutput, lastPage bool) bool {
		dbParams = append(dbParams, page.Parameters...)
		// if items returned < default MaxItems
		return !(len(page.Parameters) < 100)
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "rds: failed to describe db parameters")
	}

	var updateParameters []*rds.Parameter
	var resetParameters []*rds.Parameter
	// Compare current vs spec vs default, modify if needed
	for _, parameter := range dbParams {
		val, ok := instance.Spec.Parameters[aws.StringValue(parameter.ParameterName)]
		// if parameter in spec
		if ok {
			// if spec does not match current param
			if val != aws.StringValue(parameter.ParameterValue) {
				newParam := &rds.Parameter{
					ParameterName:  parameter.ParameterName,
					ParameterValue: aws.String(val),
					ApplyMethod:    parameter.ApplyMethod,
				}
				updateParameters = append(updateParameters, newParam)
			}
			continue
		}
		// If the value is not in spec make sure it matches expected default value
		for _, defaultParameter := range defaultDBParams {
			if aws.StringValue(defaultParameter.ParameterName) == aws.StringValue(parameter.ParameterName) {
				if aws.StringValue(defaultParameter.ParameterValue) != aws.StringValue(parameter.ParameterValue) {
					// Can only reset 20 parameters at a time.
					if len(resetParameters) < 20 {
						resetParameters = append(resetParameters, defaultParameter)
					} else {
						// No point in continuing if we can't modify more
						break
					}
				}
				break
			}
		}
	}

	if len(updateParameters) > 0 {
		_, err = comp.rdsAPI.ModifyDBParameterGroup(&rds.ModifyDBParameterGroupInput{
			DBParameterGroupName: aws.String(instance.Name),
			Parameters:           updateParameters,
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok && aerr.Code() == rds.ErrCodeInvalidDBParameterGroupStateFault {
				// Not returning error to retain RequeueAfter behavior.
				return components.Result{RequeueAfter: time.Second * 30}, nil
			}
			return components.Result{}, errors.Wrap(err, "rds: unable to modify db parameter group")
		}
		return components.Result{RequeueAfter: time.Second * 30}, nil
	}

	if len(resetParameters) > 0 {
		_, err := comp.rdsAPI.ResetDBParameterGroup(&rds.ResetDBParameterGroupInput{
			DBParameterGroupName: aws.String(instance.Name),
			Parameters:           resetParameters,
			ResetAllParameters:   aws.Bool(false),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok && aerr.Code() == rds.ErrCodeInvalidDBParameterGroupStateFault {
				// Not returning error to retain RequeueAfter behavior.
				return components.Result{RequeueAfter: time.Second * 30}, nil
			}
			return components.Result{}, errors.Wrap(err, "rds: failed to reset db parameter group")
		}
		return components.Result{RequeueAfter: time.Second * 30}, nil
	}

	return components.Result{}, nil
}

func (comp *dbParameterGroupComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)
	describeDBParameterGroupsOutput, err := comp.rdsAPI.DescribeDBParameterGroups(&rds.DescribeDBParameterGroupsInput{
		DBParameterGroupName: aws.String(instance.Name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == rds.ErrCodeDBParameterGroupNotFoundFault {
			return components.Result{}, nil
		}
		return components.Result{}, errors.Wrap(err, "rds: failed to describe parameter group for finalizer")
	}

	_, err = comp.rdsAPI.DeleteDBParameterGroup(&rds.DeleteDBParameterGroupInput{
		DBParameterGroupName: describeDBParameterGroupsOutput.DBParameterGroups[0].DBParameterGroupName,
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to delete parameter group for finalizer")
	}

	// Our parameter group is in the process of being deleted
	return components.Result{}, nil
}
