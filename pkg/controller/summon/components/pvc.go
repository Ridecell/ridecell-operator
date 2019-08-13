/*
Copyright 2018-2019 Ridecell, Inc.

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

type pvcComponent struct {
	templatePath string
}

func NewPVC(templatePath string) *pvcComponent {
	return &pvcComponent{templatePath: templatePath}
}

func (comp *pvcComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.PersistentVolumeClaim{},
	}
}

func (comp *pvcComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *pvcComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*corev1.PersistentVolumeClaim)
		existing := existingObj.(*corev1.PersistentVolumeClaim)
		// Copy the Spec over.
		existing.Spec.Resources.Requests = goal.Spec.Resources.Requests
		existing.Spec.AccessModes = goal.Spec.AccessModes
		existing.Spec.StorageClassName = goal.Spec.StorageClassName
		return nil
	})
	return res, err
}
