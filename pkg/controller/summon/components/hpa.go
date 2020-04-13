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
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/runtime"
)

type hpaComponent struct {
	templatePath string
}

func NewHPA(templatePath string) *hpaComponent {
	return &hpaComponent{templatePath: templatePath}
}

func (comp *hpaComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&autoscalingv2beta2.HorizontalPodAutoscaler{},
	}
}

func (_ *hpaComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *hpaComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*autoscalingv2beta2.HorizontalPodAutoscaler)
		existing := existingObj.(*autoscalingv2beta2.HorizontalPodAutoscaler)
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
