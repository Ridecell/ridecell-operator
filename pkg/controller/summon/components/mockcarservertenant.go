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
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type newMockCarServerTenantComponent struct{}

func NewMockCarServerTenant() *newMockCarServerTenantComponent {
	return &newMockCarServerTenantComponent{}
}

func (_ *newMockCarServerTenantComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&summonv1beta1.MockCarServerTenant{},
	}
}

func (_ *newMockCarServerTenantComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *newMockCarServerTenantComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	if !instance.Spec.EnableMockCarServer {
		// Check if mock tenant is present or not?
		// if present, then delete it
		mockcarservertenant := &summonv1beta1.MockCarServerTenant{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, mockcarservertenant)
		if err == nil {
			err = ctx.Delete(ctx.Context, mockcarservertenant)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "mockcarservertenant: unable to delete MockCarServerTenant %s", instance.Name)
			}
		}
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate("mockcarservertenant.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*summonv1beta1.MockCarServerTenant)
		existing := existingObj.(*summonv1beta1.MockCarServerTenant)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
