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

package components_test

import (
	"database/sql"
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	pecomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresextension/components"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("PostgresExtension Database Component", func() {
	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=foo-database port=5432 dbname=summon user=admin password='secretdbpass' sslmode=require", db)

		// Baseline test instance.
		instance.Spec = dbv1beta1.PostgresExtensionSpec{
			ExtensionName: "postgis",
			Database: dbv1beta1.PostgresConnection{
				Host:     "foo-database",
				Port:     5432,
				Username: "admin",
				Database: "summon",
				PasswordSecretRef: apihelpers.SecretRef{
					Name: "admin.foo-database.credentials",
					Key:  "password",
				},
			},
		}
		// Password secret.
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "admin.foo-database.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("secretdbpass"),
			},
		}
		ctx.Client = fake.NewFakeClient(dbSecret)
	})

	AfterEach(func() {
		db.Close()
		dbpool.Dbs.Delete("postgres host=foo-database port=5432 dbname=summon user=admin password='secretdbpass' sslmode=require")

		// Check for any unmet expectations.
		err := dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfulfilled database expectations: %s", err))
		}
	})

	It("creates an extension", func() {
		dbMock.ExpectExec("CREATE EXTENSION IF NOT EXISTS \"postgis\"").WithArgs().WillReturnResult(sqlmock.NewResult(0, 1))
		dbMock.ExpectExec("ALTER EXTENSION \"postgis\" UPDATE").WithArgs().WillReturnResult(sqlmock.NewResult(0, 1))

		comp := pecomponents.NewDatabase()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
		Expect(instance.Status.Message).To(Equal("Extension postgis created"))
	})

})
