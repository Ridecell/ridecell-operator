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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"strings"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Don't create object when associated component is not active, delete if already exists
	if strings.HasPrefix(comp.templatePath, "businessPortal") && *instance.Spec.Replicas.BusinessPortal == 0 {
		return components.Result{}, comp.deleteObject(ctx, instance, "businessportal")
	} else if strings.HasPrefix(comp.templatePath, "tripShare") && *instance.Spec.Replicas.TripShare == 0 {
		return components.Result{}, comp.deleteObject(ctx, instance, "tripshare")
	} else if strings.HasPrefix(comp.templatePath, "pulse") && *instance.Spec.Replicas.Pulse == 0 {
		return components.Result{}, comp.deleteObject(ctx, instance, "pulse")
	} else if strings.HasPrefix(comp.templatePath, "dispatch") && *instance.Spec.Replicas.Dispatch == 0 {
		return components.Result{}, comp.deleteObject(ctx, instance, "dispatch")
	} else if strings.HasPrefix(comp.templatePath, "hwAux") && *instance.Spec.Replicas.HwAux == 0 {
		return components.Result{}, comp.deleteObject(ctx, instance, "hwaux")
	} else if strings.HasPrefix(comp.templatePath, "customerportal") && *instance.Spec.Replicas.CustomerPortal == 0 {
		return components.Result{}, comp.deleteObject(ctx, instance, "customerportal")
	}

	requeue := false
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*policyv1beta1.PodDisruptionBudget)
		existing := existingObj.(*policyv1beta1.PodDisruptionBudget)
		// This comparison and deletion section is a temporary hack until kubernetes 1.15
		// where pod disruption budgets aren't immuatable
		if !reflect.DeepEqual(goal.Spec, existing.Spec) {
			err := ctx.Client.Delete(ctx.Context, existing)
			if err != nil && !kerrors.IsNotFound(err) {
				return errors.Wrap(err, "disruptionbudget: failed to delete disruption budget")
			}
			requeue = true
			return nil
		}
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})

	// This update prevents a notfound error on update, also part of the temporary hack
	if requeue {
		return components.Result{Requeue: true}, nil
	}
	return res, err
}

func (_ *podDisruptionBudgetComponent) deleteObject(ctx *components.ComponentContext, instance *summonv1beta1.SummonPlatform, componentName string) error {
	obj := &policyv1beta1.PodDisruptionBudget{}

	err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: instance.Name + "-" + componentName, Namespace: instance.Namespace}, obj)
	if err == nil {
		err = ctx.Delete(ctx.Context, obj)
		if err != nil {
			return errors.Wrapf(err, "failed to delete existing "+componentName)
		}
	} else if err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrapf(err, "failed to get and delete existing "+componentName)
	}
	return nil
}
