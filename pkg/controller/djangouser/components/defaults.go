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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type defaultsComponent struct{}

func NewDefaults() *defaultsComponent {
	return &defaultsComponent{}
}

func (_ *defaultsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *defaultsComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *defaultsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.DjangoUser)

	// Fill in defaults.
	if instance.Spec.Email == "" {
		// Use the object name by default, replacing the first `.` with and `@` because `@` isn't allowed in object names.
		instance.Spec.Email = strings.Replace(instance.Name, ".", "@", 1)
	}
	if instance.Spec.Database.Port == 0 {
		// Default Postgres port.
		instance.Spec.Database.Port = 5432
	}
	if instance.Spec.Database.PasswordSecretRef.Key == "" {
		// Use "password" as the default key.
		instance.Spec.Database.PasswordSecretRef.Key = "password"
	}
	if instance.Spec.Superuser {
		// Superuser means everything else too implicitly.
		instance.Spec.Staff = true
		instance.Spec.Manager = true
		instance.Spec.Dispatcher = true
	}
	if instance.Spec.Staff || instance.Spec.Manager || instance.Spec.Dispatcher {
		// You are active if you have perms. This might be wrong if we want to disable user but leave their perms?
		instance.Spec.Active = true
	}

	return components.Result{}, nil
}
