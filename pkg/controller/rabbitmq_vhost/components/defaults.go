/*
Copyright 2018 Ridecell, Inc.

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
	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/apimachinery/pkg/runtime"
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
	instance := ctx.Top.(*dbv1beta1.RabbitmqVhost)

	// Fill in defaults.
	if instance.Spec.VhostName == "" {
		// Default extension name is just the name of the resource.
		instance.Spec.VhostName = instance.Name
	}
	if instance.Spec.Connection.PasswordSecretRef.Key == "" {
		// Use "guest" as the default key.
		instance.Spec.Connection.PasswordSecretRef.Key = "guest"
	}
	if instance.Spec.Connection.Username == "" {
		// Use "guest" as the default username.
		instance.Spec.Connection.Username = "guest"
	}
	if instance.Spec.ClusterHost == "" {
		//Use localhost as the default clusterhost
		instance.Spec.ClusterHost = "http://127.0.0.1:15672"
	}

	return components.Result{}, nil
}
