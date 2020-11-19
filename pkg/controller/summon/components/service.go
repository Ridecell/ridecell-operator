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
	"strings"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type serviceComponent struct {
	templatePath string
}

func NewService(templatePath string) *serviceComponent {
	return &serviceComponent{templatePath: templatePath}
}

func (comp *serviceComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Service{},
	}
}

func (_ *serviceComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// Services have no dependencies, always reconcile.
	return true
}

func (comp *serviceComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Don't create service when associated component is not active
	if strings.HasPrefix(comp.templatePath, "redis") && instance.Spec.MigrationOverrides.RedisHostname != "" {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "businessPortal") && *instance.Spec.Replicas.BusinessPortal == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "tripShare") && *instance.Spec.Replicas.TripShare == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "pulse") && *instance.Spec.Replicas.Pulse == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "dispatch") && *instance.Spec.Replicas.Dispatch == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "hwAux") && *instance.Spec.Replicas.HwAux == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "celerybeat") && instance.Spec.UseCeleryRedBeat {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "celeryredbeat") && !(instance.Spec.UseCeleryRedBeat) {
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*corev1.Service)
		existing := existingObj.(*corev1.Service)
		// Special case: Services mutate the ClusterIP value in the Spec and it should be preserved.
		goal.Spec.ClusterIP = existing.Spec.ClusterIP
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
