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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
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
	instance := ctx.Top.(*dbv1beta1.DbConfig)

	// Check for an invalid configuration.
	if instance.Spec.Postgres.RDS == nil && instance.Spec.Postgres.Local == nil {
		return components.Result{}, errors.New("Must specify RDS or Local postgres configuration")
	} else if instance.Spec.Postgres.RDS != nil && instance.Spec.Postgres.Local != nil {
		return components.Result{}, errors.New("Cannot specify both RDS and Local postgres configuration")
	}

	// Fill in defaults.
	if instance.Spec.CreatePeriscopeUser == nil {
		val := true
		instance.Spec.CreatePeriscopeUser = &val
	}

	return components.Result{}, nil
}
