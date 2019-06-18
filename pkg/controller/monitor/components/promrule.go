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
	"fmt"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
)

type promrulComponent struct {
}

func NewPromrule() *promrulComponent {
	return &promrulComponent{}
}

func (_ *promrulComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *promrulComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *promrulComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	fmt.Println("in promrule")
	instance := ctx.Top.(*monitoringv1beta1.Monitor)
	rules, _ := yaml.Marshal(instance.Spec.MetricAlertRules)
	extras := map[string]interface{}{}
	extras["alerts"] = string(rules)
	res, _, err := ctx.CreateOrUpdate("prometheus_rule.yml.tpl", extras, func(_goalObj, existingObj runtime.Object) error {
		_ = existingObj.(*monitoringv1beta1.Monitor)
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to create PrometheusRule for %s", instance.Name)
	}
	return res, nil
}
