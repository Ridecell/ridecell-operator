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

// TODO: This whole thing should probably be its own custom resource.

import (
	"k8s.io/apimachinery/pkg/runtime"

	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type pullSecretComponent struct {
	templatePath string
}

func NewPullSecret(templatePath string) *pullSecretComponent {
	return &pullSecretComponent{templatePath: templatePath}
}

func (comp *pullSecretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&secretsv1beta1.PullSecret{},
	}
}

func (_ *pullSecretComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// Secrets have no dependencies, always reconcile.
	return true
}

func (comp *pullSecretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var existing *secretsv1beta1.PullSecret
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*secretsv1beta1.PullSecret)
		existing = existingObj.(*secretsv1beta1.PullSecret)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	res.StatusModifier = func(obj runtime.Object) error {
		instance := obj.(*summonv1beta1.SummonPlatform)
		instance.Status.PullSecretStatus = existing.Status.Status
		return nil
	}
	return res, err
}
