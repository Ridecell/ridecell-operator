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
	"k8s.io/apimachinery/pkg/runtime"

	policyv1beta1 "k8s.io/api/policy/v1beta1"
)

type podDisruptionBudgetComponent struct {
	templatePath string
}

func NewPodDisruptionBudget(templatePath string) *podDisruptionBudgetComponent {
	return &podDisruptionBudgetComponent{
		templatePath: templatePath,
	}
}

func (comp *podDisruptionBudgetComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&policyv1beta1.PodDisruptionBudget{},
	}
}

func (_ *podDisruptionBudgetComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *podDisruptionBudgetComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*policyv1beta1.PodDisruptionBudget)
		existing := existingObj.(*policyv1beta1.PodDisruptionBudget)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
