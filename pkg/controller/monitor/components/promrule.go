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
	pomonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
)

type promruleComponent struct {
}

func NewPromrule() *promruleComponent {
	return &promruleComponent{}
}

func (_ *promruleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *promruleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *promruleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*monitoringv1beta1.Monitor)
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
