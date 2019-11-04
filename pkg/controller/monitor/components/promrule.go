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

	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	pomonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const promruleFinalizer = "finalizer.promrule.monitoring.ridecell.io"

type promruleComponent struct {
}

func NewPromrule() *promruleComponent {
	return &promruleComponent{}
}

func (_ *promruleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&monitoringv1beta1.Monitor{},
	}
}

func (_ *promruleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *promruleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*monitoringv1beta1.Monitor)

	// absence MetricAlertRules should not retrun error else other components will break
	if len(instance.Spec.MetricAlertRules) <= 0 {
		return components.Result{}, nil
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(promruleFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(promruleFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(promruleFinalizer, instance) {
			if !instance.Spec.SkipFinalizers {
				promrule := &pomonitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance.Name,
						Namespace: instance.Namespace,
					}}
				err := ctx.Delete(ctx.Context, promrule)
				if err != nil {
					if !k8serr.IsNotFound(err) {
						return components.Result{}, errors.Wrapf(err, "failed to delete PrometheusRule ")
					}
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(promruleFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update PrometheusRule while removing finalizer")
			}
		}
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate("prometheus_rule.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*pomonitoringv1.PrometheusRule)
		existing := existingObj.(*pomonitoringv1.PrometheusRule)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to create PrometheusRule for %s", instance.Name)
	}
	return res, nil
}
