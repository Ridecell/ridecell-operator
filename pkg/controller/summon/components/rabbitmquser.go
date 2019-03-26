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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type rabbitmqUserComponent struct {
	templatePath string
}

func NewRabbitmqUser(templatePath string) *rabbitmqUserComponent {
	return &rabbitmqUserComponent{templatePath: templatePath}
}

func (comp *rabbitmqUserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.RabbitmqUser{},
	}
}

func (_ *rabbitmqUserComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// Has no dependencies, always reconcilable.
	return true
}

func (comp *rabbitmqUserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	var existing *dbv1beta1.RabbitmqUser
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.RabbitmqUser)
		existing = existingObj.(*dbv1beta1.RabbitmqUser)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
