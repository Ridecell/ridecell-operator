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
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type ingressComponent struct {
	templatePath string
}

func NewIngress(templatePath string) *ingressComponent {
	return &ingressComponent{templatePath: templatePath}
}

func (comp *ingressComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&extv1beta1.Ingress{},
	}
}

func (_ *ingressComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *ingressComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Don't create ingress when associated component is not active (businessPortal, pulse, tripshare)
	if strings.HasPrefix(comp.templatePath, "businessPortal") && *instance.Spec.Replicas.BusinessPortal == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "tripShare") && *instance.Spec.Replicas.TripShare == 0 {
		return components.Result{}, nil
	} else if strings.HasPrefix(comp.templatePath, "pulse") && *instance.Spec.Replicas.Pulse == 0 {
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*extv1beta1.Ingress)
		existing := existingObj.(*extv1beta1.Ingress)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
