/*
Copyright 2020 Ridecell, Inc.

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
	"regexp"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	crossplanev1alpha1 "github.com/crossplane/provider-aws/apis/identity/v1alpha1"
	crossplanev1beta1 "github.com/crossplane/provider-aws/apis/identity/v1beta1"

	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type iamRoleComponent struct {
	templatePath string
}

func NewIAMRole(templatePath string) *iamRoleComponent {
	return &iamRoleComponent{templatePath: templatePath}
}

func (comp *iamRoleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&crossplanev1beta1.IAMRole{},
	}
}

func (_ *iamRoleComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	//instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// Check on the UseIAM Role flag
	return true
}

func (comp *iamRoleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	permissionsBoundaryArn := os.Getenv("PERMISSIONS_BOUNDARY_ARN")
	if permissionsBoundaryArn == "" {
		return components.Result{}, errors.Errorf("iamrole: permissions_boundary_arn is empty")
	}
	match := regexp.MustCompile(`:([0-9]{6,}):`).FindStringSubmatch(permissionsBoundaryArn)
	if match == nil {
		return components.Result{}, errors.Errorf("iamrole: unable to get account id from boundary arn")
	}
	accountID := match[1]

	// Data to be copied over to template
	extra := map[string]interface{}{}
	extra["permissionsBoundaryArn"] = permissionsBoundaryArn
	extra["accountId"] = accountID
	extra["optimusBucketName"] = instance.Spec.OptimusBucketName
	extra["mivBucket"] = fmt.Sprintf("ridecell-%s-miv", instance.Name)
	extra["assumeRolePolicyDocument"] = os.Getenv("AWS_ASSUME_ROLE_POLICY_DOCUMENT")
	if instance.Spec.MIV.ExistingBucket != "" {
		extra["mivBucket"] = instance.Spec.MIV.ExistingBucket
	}

	// Create IAMPolicy
	_, _, err := ctx.CreateOrUpdate("aws/iampolicy.yml.tpl", extra, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*crossplanev1alpha1.IAMPolicy)
		existing := existingObj.(*crossplanev1alpha1.IAMPolicy)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Errorf("iamrole: unable to create/update iampolicy")
	}

	// Create IAMRole
	_, _, err = ctx.CreateOrUpdate(comp.templatePath, extra, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*crossplanev1beta1.IAMRole)
		existing := existingObj.(*crossplanev1beta1.IAMRole)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Errorf("iamrole: unable to create/update iamrole")
	}

	// Create IAMRolePolicyAttachment
	res, _, err := ctx.CreateOrUpdate("aws/iamrolepolicyattachment.yml.tpl", extra, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*crossplanev1beta1.IAMRolePolicyAttachment)
		existing := existingObj.(*crossplanev1beta1.IAMRolePolicyAttachment)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})

	return res, err
}
