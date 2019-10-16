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
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

type serviceMonitorComponent struct {
	templatePath string
}

func NewServiceMonitor(templatePath string) *serviceMonitorComponent {
	return &serviceMonitorComponent{templatePath: templatePath}
}

func (comp *serviceMonitorComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (comp *serviceMonitorComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *serviceMonitorComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// If our flag is not set or is false exit early
	if instance.Spec.Metrics.Web == nil || !*instance.Spec.Metrics.Web {
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*promv1.ServiceMonitor)
		existing := existingObj.(*promv1.ServiceMonitor)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, errors.Wrapf(err, "servicemonitor: failed to update template %s", comp.templatePath)
	}
	return components.Result{}, nil
}
