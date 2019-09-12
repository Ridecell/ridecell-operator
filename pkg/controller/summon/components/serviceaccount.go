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

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type serviceAccountComponent struct{}

func NewServiceAccount() *serviceAccountComponent {
	return &serviceAccountComponent{}
}

func (comp *serviceAccountComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&gcpv1beta1.ServiceAccount{},
	}
}

func (_ *serviceAccountComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	return instance.Spec.GCPProject != ""
}

func (comp *serviceAccountComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.CreateOrUpdate("gcp/serviceaccount.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*gcpv1beta1.ServiceAccount)
		existing := existingObj.(*gcpv1beta1.ServiceAccount)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
