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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type statusComponent struct{}

func NewStatus() *statusComponent {
	return &statusComponent{}
}

func (_ *statusComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *statusComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *statusComponent) Reconcile(_ *components.ComponentContext) (components.Result, error) {
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.DbConfig)
		if instance.Spec.Postgres.Mode == "Shared" && instance.Status.Postgres.Status != "Ready" {
			return nil
		}
		instance.Status.Status = dbv1beta1.StatusReady
		// TODO a better message
		instance.Status.Message = "Database ready"
		return nil
	}}, nil
}
