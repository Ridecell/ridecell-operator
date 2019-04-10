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
	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/apimachinery/pkg/runtime"
)

type userComponent struct{}

func NewUser() *userComponent {
	return &userComponent{}
}

func (_ *userComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.RabbitmqUser{},
	}
}

func (_ *userComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*dbv1beta1.RabbitmqVhost)
	return !instance.Spec.SkipUser
}

func (comp *userComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.CreateOrUpdate("rabbitmq_user.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.RabbitmqUser)
		existing := existingObj.(*dbv1beta1.RabbitmqUser)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
