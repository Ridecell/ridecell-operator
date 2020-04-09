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
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/apimachinery/pkg/runtime"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
)

type vpaComponent struct {
	templatePath string
}

func NewVPA(templatePath string) *vpaComponent {
	return &vpaComponent{templatePath: templatePath}
}

func (comp *vpaComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&autoscalingv1.VerticalPodAutoscaler{},
	}
}

func (_ *vpaComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *vpaComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*autoscalingv1.VerticalPodAutoscaler)
		existing := existingObj.(*autoscalingv1.VerticalPodAutoscaler)
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
