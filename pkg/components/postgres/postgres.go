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

package postgres

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
)

// Open a connection to the Postgres database as defined by a PostgresConnection object.
func Open(ctx *components.ComponentContext, dbInfo *dbv1beta1.PostgresConnection) (*sql.DB, error) {
	dbPassword, err := dbInfo.PasswordSecretRef.Resolve(ctx, "password")
	if err != nil {
		return nil, errors.Wrap(err, "unable to resolve secret")
	}
	// A default port.
	port := dbInfo.Port
	if port == 0 {
		port = 5432
	}
	sslmode := "require"
	if dbInfo.SSLMode != "" {
		sslmode = dbInfo.SSLMode
	}
	connStr := fmt.Sprintf("host=%s port=%v dbname=%s user=%v password='%s' sslmode=%s", dbInfo.Host, uint16(port), dbInfo.Database, dbInfo.Username, dbPassword, sslmode)
	db, err := dbpool.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "database: Unable to open database connection")
	}
	return db, nil
}
