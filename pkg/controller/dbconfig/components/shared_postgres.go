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
	"fmt"

	"github.com/pkg/errors"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type sharedPostgresComponent struct{}

func NewSharedPostgres() *sharedPostgresComponent {
	return &sharedPostgresComponent{}
}

func (_ *sharedPostgresComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *sharedPostgresComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*dbv1beta1.DbConfig)
	return instance.Spec.Postgres.Mode == "Shared"
}

func (comp *sharedPostgresComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.DbConfig)

	if instance.Spec.Postgres.RDS != nil {
		return comp.reconcileRDS(ctx)
	} else if instance.Spec.Postgres.Local != nil {
		return comp.reconcileLocal(ctx)
	}

	return components.Result{}, errors.New("unexpected error, shared database is not a known type")
}

func (comp *sharedPostgresComponent) reconcileRDS(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.DbConfig)

	var existing *dbv1beta1.RDSInstance
	res, _, err := ctx.CreateOrUpdate("rds.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing = existingObj.(*dbv1beta1.RDSInstance)
		existing.Spec = *instance.Spec.Postgres.RDS
		return nil
	})

	if err != nil {
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.DbConfig)
			instance.Status.Postgres.Status = existing.Status.Status
			instance.Status.Postgres.Connection = existing.Status.Connection
			return nil
		}
	}

	return res, err
}

func (comp *sharedPostgresComponent) reconcileLocal(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.DbConfig)

	var existing *postgresv1.Postgresql
	res, _, err := ctx.CreateOrUpdate("local.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing = existingObj.(*postgresv1.Postgresql)
		existing.Spec = *instance.Spec.Postgres.LocalWithDefaults(instance.Namespace)
		return nil
	})

	if err != nil {
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.DbConfig)
			instance.Status.Postgres.Status = existing.Status.String()
			instance.Status.Postgres.Connection.Host = existing.Name
			instance.Status.Postgres.Connection.Port = 5432
			instance.Status.Postgres.Connection.Username = "ridecell-admin"
			instance.Status.Postgres.Connection.PasswordSecretRef.Name = fmt.Sprintf("ridecell-admin.%s.credentials", existing.Name)
			instance.Status.Postgres.Connection.PasswordSecretRef.Key = "password"
			return nil
		}
	}

	return res, err
}
