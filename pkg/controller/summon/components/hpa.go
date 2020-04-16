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
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

type hpaComponent struct {
	templatePath string
	isAutoscaled func(*summonv1beta1.SummonPlatform) bool
}

func NewHPA(templatePath string, isAutoscaled func(*summonv1beta1.SummonPlatform) bool) *hpaComponent {
	return &hpaComponent{templatePath: templatePath, isAutoscaled: isAutoscaled}
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
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// Only reconcile if HPA autoscaling enabled. (<comp>Auto flag in ReplicaSpecs)
	if comp.isAutoscaled != nil && comp.isAutoscaled(instance) {
		res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
			goal := goalObj.(*autoscalingv2beta2.HorizontalPodAutoscaler)
			existing := existingObj.(*autoscalingv2beta2.HorizontalPodAutoscaler)
			existing.Spec = goal.Spec
			return nil
		})
		return res, err
	}
	// autoscale may have been turned off. Check if HPA object is still around and delete it.
	obj, err := ctx.GetTemplate(comp.templatePath, nil)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "hpa: error rendering template %s", comp.templatePath)
	}
	hpa := obj.(*autoscalingv2beta2.HorizontalPodAutoscaler)
	err = ctx.Delete(ctx.Context, hpa)
	if err != nil && !kerrors.IsNotFound(err) {
		// HPA object doesn't exist. Probably never enabled, or already deleted.
		return components.Result{}, errors.Wrapf(err, "hpa: error deleting existing hpa %s/%s", hpa.Namespace, hpa.Name)
	}
	return components.Result{}, nil
}
