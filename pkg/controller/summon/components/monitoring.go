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
	"k8s.io/apimachinery/pkg/runtime"

	rmonitor "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type monitoringComponent struct{}

func NewMonitoring() *monitoringComponent {
	return &monitoringComponent{}
}

func (_ *monitoringComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&rmonitor.Monitor{},
	}
}

func (_ *monitoringComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *monitoringComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	if !instance.Spec.Monitoring.Enabled || len(instance.Spec.Notifications.SlackChannel) == 0 {
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate("monitoring.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*rmonitor.Monitor)
		existing := existingObj.(*rmonitor.Monitor)
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
