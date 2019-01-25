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
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type iamUserComponent struct {
	templatePath string
}

func NewIAMUser(templatePath string) *iamUserComponent {
	return &iamUserComponent{templatePath: templatePath}
}

func (comp *iamUserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&awsv1beta1.IAMUser{},
	}
}

func (_ *iamUserComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// Secrets have no dependencies, always reconcile.
	return true
}

func (comp *iamUserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var existing *awsv1beta1.IAMUser
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*awsv1beta1.IAMUser)
		existing = existingObj.(*awsv1beta1.IAMUser)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}