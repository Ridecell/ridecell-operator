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

type userComponent struct{}

func NewUser() *userComponent {
	return &userComponent{}
}

func (_ *userComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.PostgresUser{},
	}
}

func (_ *userComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)
	return instance.Status.DatabaseStatus == dbv1beta1.StatusReady && !instance.Spec.SkipUser
}

func (comp *userComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var existing *dbv1beta1.PostgresUser
	res, _, err := ctx.CreateOrUpdate("user.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.PostgresUser)
		existing = existingObj.(*dbv1beta1.PostgresUser)
		existing.Spec = goal.Spec
		return nil
	})
	res.StatusModifier = func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.PostgresDatabase)
		if existing != nil {
			instance.Status.UserStatus = existing.Status.Status
			instance.Status.Connection.Host = instance.Status.AdminConnection.Host
			instance.Status.Connection.Port = instance.Status.AdminConnection.Port
			instance.Status.Connection.SSLMode = instance.Status.AdminConnection.SSLMode
			instance.Status.Connection.Username = existing.Status.Connection.Username
			instance.Status.Connection.PasswordSecretRef = existing.Status.Connection.PasswordSecretRef
		}
		return nil
	}
	return res, err
}
