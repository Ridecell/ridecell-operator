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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

// THIS COMPONENT IS WEIRD BECAUSE IT IS USED IN BOTH THE DBCONFIG AND POSTGRESDATABASE CONTROLLERS.

type postgresComponent struct{}

func NewPostgres() *postgresComponent {
	return &postgresComponent{}
}

func (_ *postgresComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *postgresComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *postgresComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var dbconfig *dbv1beta1.DbConfig
	pqdb, ok := ctx.Top.(*dbv1beta1.PostgresDatabase)
	if ok {
		// This is a PostgresDatabase so try to load the relevant DbConfig.
		dbconfig = &dbv1beta1.DbConfig{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: pqdb.Spec.DbConfig, Namespace: pqdb.Namespace}, dbconfig)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "postgres: error getting dbconfig %s/%s for PostgresDatabase %s", pqdb.Namespace, pqdb.Spec.DbConfig, pqdb.Name)
		}
		// Do nothing in shared mode, DB is already provisioned.
		if dbconfig.Spec.Postgres.Mode == "Shared" {
			return components.Result{}, nil
		}
	} else {
		dbconfig = ctx.Top.(*dbv1beta1.DbConfig)
		// Do nothing in exclusive mode, DB will be provisioned by PostgresDatabase.
		if dbconfig.Spec.Postgres.Mode == "Exclusive" {
			return components.Result{}, nil
		}
	}

	var res components.Result
	var status string
	var conn *dbv1beta1.PostgresConnection
	var err error
	if dbconfig.Spec.Postgres.RDS != nil {
		res, status, conn, err = comp.reconcileRDS(ctx, dbconfig)
	} else if dbconfig.Spec.Postgres.Local != nil {
		res, status, conn, err = comp.reconcileLocal(ctx, dbconfig)
	} else {
		return components.Result{}, errors.New("unexpected error, postgres database is not a known type")
	}
	if err != nil {
		return res, err
	}

	if ok {
		// Updating the status for a PostgresDatabase.
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.PostgresDatabase)
			instance.Status.DatabaseStatus = status
			instance.Status.Connection = *conn
			return nil
		}
	} else {
		// Updating the status for a DbConfig.
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.DbConfig)
			instance.Status.Postgres.Status = status
			instance.Status.Postgres.Connection = *conn
			return nil
		}
	}
	return res, nil
}

func (comp *postgresComponent) reconcileRDS(ctx *components.ComponentContext, dbconfig *dbv1beta1.DbConfig) (components.Result, string, *dbv1beta1.PostgresConnection, error) {
	var existing *dbv1beta1.RDSInstance
	res, _, err := ctx.CreateOrUpdate("rds.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing = existingObj.(*dbv1beta1.RDSInstance)
		existing.Spec = *dbconfig.Spec.Postgres.RDS
		return nil
	})
	if err != nil {
		return res, "", nil, err
	}
	return res, existing.Status.Status, &existing.Status.Connection, err
}

func (comp *postgresComponent) reconcileLocal(ctx *components.ComponentContext, dbconfig *dbv1beta1.DbConfig) (components.Result, string, *dbv1beta1.PostgresConnection, error) {
	var existing *postgresv1.Postgresql
	res, _, err := ctx.CreateOrUpdate("local.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing = existingObj.(*postgresv1.Postgresql)
		existing.Spec = *dbconfig.Spec.Postgres.Local.DeepCopy()
		if existing.Spec.TeamID == "" {
			instanceMeta := ctx.Top.(metav1.Object)
			existing.Spec.TeamID = instanceMeta.GetName()
		}
		if existing.Spec.NumberOfInstances == 0 {
			existing.Spec.NumberOfInstances = 2
		}
		if existing.Spec.Users == nil {
			existing.Spec.Users = map[string]postgresv1.UserFlags{}
		}
		existing.Spec.Users["ridecell-admin"] = postgresv1.UserFlags{"superuser"}
		return nil
	})
	if err != nil {
		return res, "", nil, err
	}
	conn := &dbv1beta1.PostgresConnection{
		Host:     existing.Name,
		Port:     5432,
		Username: "ridecell-admin",
		PasswordSecretRef: helpers.SecretRef{
			Name: fmt.Sprintf("ridecell-admin.%s.credentials", existing.Name),
			Key:  "password",
		},
	}
	return res, existing.Status.String(), conn, nil
}
