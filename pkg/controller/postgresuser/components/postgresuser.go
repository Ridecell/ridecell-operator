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

	"github.com/lib/pq"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

type PostgresUserComponent struct {
}

func NewPostgresUser() *PostgresUserComponent {
	return &PostgresUserComponent{}
}

func (_ *PostgresUserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *PostgresUserComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *PostgresUserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.PostgresUser)

	password, err := instance.Status.Connection.PasswordSecretRef.Resolve(ctx, "password")
	if err != nil {
		return components.Result{}, errors.Wrap(err, "postgres_user: failed fetch password")
	}

	db, err := postgres.Open(ctx, &instance.Spec.Connection)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "postgres_user: failed to open db connection")
	}

	// Check if user exists
	userRows, err := db.Query("SELECT usename FROM pg_user")
	if err != nil {
		return components.Result{}, errors.Wrap(err, "postgres_user: failed to query users")
	}
	defer userRows.Close()

	var existingUsers []string
	for userRows.Next() {
		var result *string
		err = userRows.Scan(&result)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "postgres_user: failed to scan row")
		}
		existingUsers = append(existingUsers, *result)
	}

	var userExists bool
	for _, existingUser := range existingUsers {
		if instance.Spec.Username == existingUser {
			userExists = true
		}
	}
	err = userRows.Err()
	if err != nil {
		return components.Result{}, errors.Wrap(err, "postgres_user: row error")
	}

	quotedUsername := pq.QuoteIdentifier(instance.Spec.Username)
	quotedPassword := utils.QuoteLiteral(password)
	// Create the user if it doesn't exist
	if !userExists {
		_, err = db.Exec(fmt.Sprintf("CREATE USER %s WITH PASSWORD %s", quotedUsername, quotedPassword))
		if err != nil {
			return components.Result{}, errors.Wrap(err, "postgres_user: failed to create database user")
		}
	}

	// Do a test query to make sure that the user is valid
	newConnection := instance.Spec.Connection.DeepCopy()
	newConnection.Database = "postgres"
	newConnection.Username = instance.Spec.Username
	newConnection.PasswordSecretRef = instance.Status.Connection.PasswordSecretRef

	testdb, err := postgres.Open(ctx, newConnection)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "postgres_user: failed to open testdb connection")
	}

	var invalidPassword bool
	noOpRows, err := testdb.Query(`SELECT 1`)
	if err != nil {
		// 28P01 == invalid password
		if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "28P01" {
			invalidPassword = true
		} else {
			return components.Result{}, errors.Wrap(err, "postgres_user: failed to query database")
		}
	} else {
		defer noOpRows.Close()
	}

	if invalidPassword {
		_, err = db.Exec(fmt.Sprintf("ALTER USER %s WITH PASSWORD %s", quotedUsername, quotedPassword))
		if err != nil {
			return components.Result{}, errors.Wrap(err, "postgres_user: failed to update user password")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.PostgresUser)
		instance.Status.Status = dbv1beta1.StatusReady
		instance.Status.Message = "User Created"
		instance.Status.Connection.Host = instance.Spec.Connection.Host
		instance.Status.Connection.Port = instance.Spec.Connection.Port
		instance.Status.Connection.SSLMode = instance.Spec.Connection.SSLMode
		instance.Status.Connection.Username = instance.Spec.Username
		return nil
	}}, nil
}
