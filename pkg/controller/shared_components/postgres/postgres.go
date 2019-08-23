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

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/pkg/errors"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
		&corev1.Service{},
		&appsv1.Deployment{},
		&monitoringv1.ServiceMonitor{},
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
	var migrationOverrides *dbv1beta1.MigrationOverridesSpec
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
				pqdb.Status.RDSInstanceID = dbconfig.Status.RDSInstanceID
				return nil
			}}, nil
		}
		migrationOverrides = &pqdb.Spec.MigrationOverrides
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
	var rdsInstanceID string
	if dbconfig.Spec.Postgres.RDS != nil {
		var rdsStatus *dbv1beta1.RDSInstanceStatus
		res, rdsStatus, conn, err = comp.reconcileRDS(ctx, dbconfig, migrationOverrides)
		status = rdsStatus.Status
		rdsInstanceID = rdsStatus.InstanceID

	} else if dbconfig.Spec.Postgres.Local != nil {
		res, status, conn, err = comp.reconcileLocal(ctx, dbconfig)
	} else {
		return components.Result{}, errors.New("unexpected error, postgres database is not a known type")
	}
	if err != nil {
		return res, err
	}
	_, err = comp.reconcileExporter(ctx, conn)
	if err != nil {
		return res, errors.Wrap(err, "error while reconciling exporter deployment")
	}
	_, err = comp.reconcileService(ctx)
	if err != nil {
		return res, errors.Wrap(err, "error while reconciling exporter service")
	}
	_, err = comp.reconcileServiceMonitor(ctx)
	if err != nil {
		return res, errors.Wrap(err, "error while reconciling exporter service monitor")
	}
	res, err = comp.reconcilePeriscopeUser(ctx, dbconfig, conn)
	if err != nil {
		return res, errors.Wrap(err, "error while reconciling periscope postgres user")
	}

	if comp.mode == "Exclusive" {
		// Updating the status for a PostgresDatabase.
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.PostgresDatabase)
			instance.Status.DatabaseClusterStatus = status
			instance.Status.AdminConnection = *conn
			instance.Status.RDSInstanceID = rdsInstanceID
			return nil
		}
	} else {
		// Updating the status for a DbConfig.
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.DbConfig)
			instance.Status.Postgres.Status = status
			instance.Status.Postgres.Connection = *conn
			instance.Status.RDSInstanceID = rdsInstanceID
			return nil
		}
	}
	return res, nil
}

func (comp *postgresComponent) reconcileRDS(ctx *components.ComponentContext, config *dbv1beta1.DbConfig, migrationOverrides *dbv1beta1.MigrationOverridesSpec) (components.Result, *dbv1beta1.RDSInstanceStatus, *dbv1beta1.PostgresConnection, error) {
	var existing *dbv1beta1.RDSInstance
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("rds.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing = existingObj.(*dbv1beta1.RDSInstance)
		existing.Spec = *config.Spec.Postgres.RDS
		if migrationOverrides != nil {
			if migrationOverrides.RDSInstanceID != "" {
				existing.Spec.InstanceID = migrationOverrides.RDSInstanceID
			}
			if migrationOverrides.RDSMasterUsername != "" {
				existing.Spec.Username = migrationOverrides.RDSMasterUsername
			}
		}
		return nil
	})
	if err != nil {
		return res, nil, nil, err
	}
	return res, &existing.Status, &existing.Status.Connection, err
}

func (comp *postgresComponent) reconcileLocal(ctx *components.ComponentContext, dbconfig *dbv1beta1.DbConfig) (components.Result, string, *dbv1beta1.PostgresConnection, error) {
	var existing *postgresv1.Postgresql
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("local.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing = existingObj.(*postgresv1.Postgresql)
		// Copy over fields.
		local := dbconfig.Spec.Postgres.Local
		existing.Spec.PostgresqlParam.PgVersion = local.PostgresqlParam.PgVersion
		existing.Spec.PostgresqlParam.Parameters = local.PostgresqlParam.Parameters
		existing.Spec.Volume = local.Volume
		// existing.Spec.Patroni = local.Patroni
		existing.Spec.Resources = local.Resources
		existing.Spec.DockerImage = local.DockerImage
		existing.Spec.EnableMasterLoadBalancer = local.EnableMasterLoadBalancer
		existing.Spec.EnableReplicaLoadBalancer = local.EnableReplicaLoadBalancer
		existing.Spec.AllowedSourceRanges = local.AllowedSourceRanges
		existing.Spec.NumberOfInstances = local.NumberOfInstances
		existing.Spec.Users = local.Users
		existing.Spec.MaintenanceWindows = local.MaintenanceWindows
		existing.Spec.Clone = local.Clone
		existing.Spec.Databases = local.Databases
		existing.Spec.Tolerations = local.Tolerations
		existing.Spec.Sidecars = local.Sidecars
		existing.Spec.PodPriorityClassName = local.PodPriorityClassName
		// existing.Spec.InitContainers = local.InitContainers // Newer version of postgres-operator?
		// existing.Spec.ShmVolume = local.ShmVolume
		// Standard fields
		instanceMeta := ctx.Top.(metav1.Object)
		existing.Spec.TeamID = instanceMeta.GetName()
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
		Database: "postgres",
	}
	return res, existing.Status.String(), conn, nil
}

func (comp *postgresComponent) reconcileExporter(ctx *components.ComponentContext, conn *dbv1beta1.PostgresConnection) (components.Result, error) {
	extras := map[string]interface{}{}
	extras["Conn"] = conn
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("postgres-exporter.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
		existing := existingObj.(*appsv1.Deployment)
		goal := goalObj.(*appsv1.Deployment)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, err
}

func (comp *postgresComponent) reconcileService(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("postgres-service.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Service)
		goal := goalObj.(*corev1.Service)
		// Special case: Services mutate the ClusterIP value in the Spec and it should be preserved.
		goal.Spec.ClusterIP = existing.Spec.ClusterIP
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, err
}

func (comp *postgresComponent) reconcileServiceMonitor(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("service-monitor.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		existing := existingObj.(*monitoringv1.ServiceMonitor)
		goal := goalObj.(*monitoringv1.ServiceMonitor)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, err
}

func (comp *postgresComponent) reconcilePeriscopeUser(ctx *components.ComponentContext, dbconfig *dbv1beta1.DbConfig, conn *dbv1beta1.PostgresConnection) (components.Result, error) {
	var existing *dbv1beta1.PostgresUser
	extras := map[string]interface{}{}
	extras["Conn"] = conn
	if !dbconfig.Spec.NoCreatePeriscopeUser {
		res, _, err := ctx.WithTemplates(Templates).CreateOrUpdate("periscopeuser.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
			existing = existingObj.(*dbv1beta1.PostgresUser)
			goal := goalObj.(*dbv1beta1.PostgresUser)
			existing.Spec = goal.Spec
			return nil
		})
		if err != nil {
			return res, err
		}
		return res, err
	} else {
		// Check if periscope user already exists and delete it
		user, err := ctx.WithTemplates(Templates).GetTemplate("periscopeuser.yml.tpl", extras)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "unable to load periscope.yml.tpl to delete existing periscope user")
		}
		err = ctx.Delete(ctx.Context, user)
		if err != nil && !kerrors.IsNotFound(err) {
			return components.Result{Requeue: true}, errors.Wrap(err, "unable to find and delete periscopeuser")
		}
		return components.Result{}, nil
	}
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
