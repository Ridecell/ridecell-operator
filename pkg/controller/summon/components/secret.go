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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type secretComponent struct {
	templatePath string
	limitFn      func(*components.ComponentContext) bool
}

func NewSecret(templatePath string, fn func(*components.ComponentContext) bool) *secretComponent {
	return &secretComponent{templatePath: templatePath, limitFn: fn}
}

func (comp *secretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
	}
}

func (comp *secretComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	if comp.limitFn != nil {
		return comp.limitFn(ctx)
	} else {
		return true
	}
}

func (comp *secretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*corev1.Secret)
		existing := existingObj.(*corev1.Secret)
		if goal.Data == nil && goal.StringData != nil {
			goal.Data = map[string][]byte{}
			for key, value := range goal.StringData {
				goal.Data[key] = []byte(value)
			}
		}
		// Copy the data over.
		existing.Data = goal.Data
		return nil
	})
	return res, err
}
