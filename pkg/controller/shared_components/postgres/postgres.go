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

package postgres

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

// THIS COMPONENT IS WEIRD BECAUSE IT IS USED IN BOTH THE DBCONFIG AND POSTGRESDATABASE CONTROLLERS.

type postgresComponent struct {
	mode string
}

func NewPostgres(mode string) *postgresComponent {
	return &postgresComponent{mode: mode}
}

func (_ *postgresComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.RDSInstance{},
		&postgresv1.Postgresql{},
		&dbv1beta1.DbConfig{},
	}
}

func (comp *postgresComponent) WatchMap(obj handler.MapObject, c client.Client) ([]reconcile.Request, error) {
	// First check if this is an owned object, if so, short circuit.
	owner := metav1.GetControllerOf(obj.Meta)
	var relevantOwner string
	if comp.mode == "Exclusive" {
		relevantOwner = "PostgresDatabase"
	} else {
		relevantOwner = "DbConfig"
	}
	if owner != nil && owner.Kind == relevantOwner {
		return []reconcile.Request{
			reconcile.Request{NamespacedName: types.NamespacedName{Name: owner.Name, Namespace: obj.Meta.GetNamespace()}},
		}, nil
	}

	requests := []reconcile.Request{}
	// If we are in PostgresDatabase, check if the change is a linked DbConfig.
	_, isDbConfig := obj.Object.(*dbv1beta1.DbConfig)
	if comp.mode == "Exclusive" && isDbConfig {
		dbs := &dbv1beta1.PostgresDatabaseList{}
		err := c.List(context.Background(), nil, dbs)
		if err != nil {
			return nil, errors.Wrap(err, "error listing postgresdatabases")
		}

		for _, db := range dbs.Items {
			// Check the DbConfig field.
			dbConfigRef := comp.dbConfigRefFor(&db)
			if dbConfigRef.Name == obj.Meta.GetName() && dbConfigRef.Namespace == obj.Meta.GetNamespace() {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: db.Name, Namespace: db.Namespace}})
			}
		}
	}

	return requests, nil
}

func (_ *postgresComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *postgresComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var dbconfig *dbv1beta1.DbConfig
	if comp.mode == "Exclusive" {
		// This is a PostgresDatabase so try to load the relevant DbConfig.
		pqdb := ctx.Top.(*dbv1beta1.PostgresDatabase)
		dbconfigRef := comp.dbConfigRefFor(pqdb)
		dbconfig = &dbv1beta1.DbConfig{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: dbconfigRef.Name, Namespace: dbconfigRef.Namespace}, dbconfig)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "postgres: error getting dbconfig %s/%s for PostgresDatabase %s", dbconfigRef.Namespace, dbconfigRef.Name, pqdb.Name)
		}
		// Do nothing in shared mode, DB is already provisioned.
		if dbconfig.Spec.Postgres.Mode == "Shared" {
			return components.Result{StatusModifier: func(obj runtime.Object) error {
				pqdb := obj.(*dbv1beta1.PostgresDatabase)
				pqdb.Status.DatabaseClusterStatus = dbconfig.Status.Postgres.Status
				pqdb.Status.AdminConnection = dbconfig.Status.Postgres.Connection
				return nil
			}}, nil
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

	if comp.mode == "Exclusive" {
		// Updating the status for a PostgresDatabase.
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.PostgresDatabase)
			instance.Status.DatabaseClusterStatus = status
			instance.Status.AdminConnection = *conn
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
	extras := map[string]interface{}{
		"DbConfig": dbconfig,
	}

	// If an override is present, insert it into the RDSSpec.
	metaObj := ctx.Top.(metav1.Object)
	id, ok := dbconfig.Spec.Postgres.RDSOverride[metaObj.GetName()]
	if ok {
		dbconfig.Spec.Postgres.RDS.InstanceID = id
	}

	var existing *dbv1beta1.RDSInstance
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("rds.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.RDSInstance)
		existing = existingObj.(*dbv1beta1.RDSInstance)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, "", nil, err
	}
	return res, existing.Status.Status, &existing.Status.Connection, err
}

func (comp *postgresComponent) reconcileLocal(ctx *components.ComponentContext, dbconfig *dbv1beta1.DbConfig) (components.Result, string, *dbv1beta1.PostgresConnection, error) {
	extras := map[string]interface{}{
		"DbConfig": dbconfig,
	}

	var existing *postgresv1.Postgresql
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("local.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*postgresv1.Postgresql)
		existing = existingObj.(*postgresv1.Postgresql)
		existing.Spec = goal.Spec
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
		Database: "postgres",
	}
	return res, existing.Status.String(), conn, nil
}

func (comp *postgresComponent) dbConfigRefFor(db *dbv1beta1.PostgresDatabase) *corev1.ObjectReference {
	name := db.Spec.DbConfigRef.Name
	if name == "" {
		name = db.Namespace
	}
	namespace := db.Spec.DbConfigRef.Namespace
	if namespace == "" {
		namespace = db.Namespace
	}
	return &corev1.ObjectReference{Name: name, Namespace: namespace}
}
