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

package components_test

import (
	"context"
	"database/sql"
	"fmt"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	"github.com/lib/pq"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	postgresusercomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresuser/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("postgresusercomponents Component", func() {
	comp := postgresusercomponents.NewPostgresUser()

	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		instance.Spec.Connection = dbv1beta1.PostgresConnection{
			Host:     "test-database",
			Port:     5432,
			Username: "test",
			PasswordSecretRef: helpers.SecretRef{
				Name: "foo-password-secret",
				Key:  "password",
			},
			Database: "test",
		}

		instance.Status.Connection.PasswordSecretRef = helpers.SecretRef{
			Name: "newPassword",
			Key:  "password",
		}

		instance.Spec.Username = "newuser"

		// password for operator user
		passwordSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-password-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"password": []byte("1234totallysecurepassword"),
			},
		}
		// password for our new user
		passwordSecret1 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "newPassword",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"password": []byte("test"),
			},
		}

		err := ctx.Client.Create(context.TODO(), passwordSecret)
		Expect(err).ToNot(HaveOccurred())
		err = ctx.Client.Create(context.TODO(), passwordSecret1)
		Expect(err).ToNot(HaveOccurred())

		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=test-database port=5432 dbname=test user=test password='1234totallysecurepassword' sslmode=require", db)
		dbpool.Dbs.Store("postgres host=test-database port=5432 dbname=postgres user=newuser password='test' sslmode=require", db)
	})

	AfterEach(func() {
		db.Close()
		dbpool.Dbs.Delete("postgres host=test-database port=5432 dbname=test user=test password='1234totallysecurepassword' sslmode=require")
		dbpool.Dbs.Delete("postgres host=test-database port=5432 dbname=postgres user=newuser password='test' sslmode=require")

		// Check for any unmet expectations.
		err := dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfulfilled database expectations: %s", err))
		}

		err = dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfufilled database expectations: %s", err))
		}
	})

	Describe("isReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("runs basic reconcile", func() {
		dbMock.ExpectQuery(`SELECT usename FROM pg_user`).WillReturnRows(sqlmock.NewRows([]string{"usename"}).AddRow("nope")).RowsWillBeClosed()
		dbMock.ExpectExec(`CREATE USER "newuser" WITH PASSWORD 'test'`).WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{"filler"}).AddRow(1)).RowsWillBeClosed()

		Expect(comp).To(ReconcileContext(ctx))
	})

	It("makes no changes", func() {
		dbMock.ExpectQuery(`SELECT usename FROM pg_user`).WillReturnRows(sqlmock.NewRows([]string{"usename"}).AddRow("newuser")).RowsWillBeClosed()
		dbMock.ExpectQuery(`SELECT 1`).WillReturnRows(sqlmock.NewRows([]string{"filler"}).AddRow(1)).RowsWillBeClosed()
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("updates the incorrect password of the new user", func() {
		dbMock.ExpectQuery(`SELECT usename FROM pg_user`).WillReturnRows(sqlmock.NewRows([]string{"usename"}).AddRow("newuser")).RowsWillBeClosed()
		dbMock.ExpectQuery(`SELECT 1`).WillReturnError(&pq.Error{Code: "28P01"})
		dbMock.ExpectExec(`ALTER USER "newuser" WITH PASSWORD 'test'`).WillReturnResult(sqlmock.NewResult(1, 1))
		Expect(comp).To(ReconcileContext(ctx))
	})

})
