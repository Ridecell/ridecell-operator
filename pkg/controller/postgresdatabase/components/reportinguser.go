/*
Copyright 2019-2020 Ridecell, Inc.

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

	"github.com/lib/pq"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

type reportingUserComponent struct{}

func NewReportingUser() *reportingUserComponent {
	return &reportingUserComponent{}
}

func (_ *reportingUserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.PostgresUser{},
	}
}

func (_ *reportingUserComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)
	// Reconcilable so long as database and reporting user is ready. If permissions already granted, don't waste time reconciling.
	return (instance.Status.DatabaseStatus == dbv1beta1.StatusReady && (instance.Status.SharedUsers.Reporting == dbv1beta1.StatusReady || instance.Status.SharedUsers.Reporting == dbv1beta1.StatusGranted))
}

func (comp *reportingUserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)

	conn := instance.Status.AdminConnection.DeepCopy()
	conn.Database = instance.Spec.DatabaseName

	db, err := postgres.Open(ctx, conn)
	if err != nil {
		return components.Result{}, err
	}

	row := db.QueryRow(`select COUNT(*) from pg_user where usename='reporting'`)
	var count int
	err = row.Scan(&count)

	if err != nil {
		return components.Result{}, errors.Wrap(err, "database: error running db check query for reporting user")
	}

	// Check if reporting was already granted permissions before.
	row = db.QueryRow(`SELECT COUNT(*) FROM information_schema.table_privileges where grantee='reporting'`)
	err = row.Scan(&count)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "database: error running db check query for reporting grants")
	}

	// if it hasn't occurred before, connect to summon instance's database and grant reporting read-permissions to public schema.
	if count == 0 {
		// Grant read access to reporting user for the postgres database.
		_, err = db.Exec("GRANT SELECT ON ALL TABLES IN SCHEMA public TO reporting")
		if err != nil {
			return components.Result{}, errors.Wrap(err, "database: error granting reporting user read-permissions")
		}

		// Grant reporting read permissions to any future tables added to public schema for the database
		_, err = db.Exec(fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s IN SCHEMA public GRANT SELECT ON TABLES TO reporting", pq.QuoteIdentifier(instance.Spec.Owner)))
		if err != nil {
			return components.Result{}, errors.Wrap(err, "database: error granting reporting user read-permissions for future public schema tables")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.PostgresDatabase)
		instance.Status.SharedUsers.Reporting = dbv1beta1.StatusGranted
		return nil
	}}, nil
}
