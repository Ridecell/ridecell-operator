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
	"database/sql"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	pdcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresdatabase/components"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("PostgresDatabase Periscope User Component", func() {
	var comp components.Component
	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		// Set up mock database.
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=mydb port=5432 dbname=foo_dev user=myuser password='postgresadminconnpw' sslmode=require", db)

		comp = pdcomponents.NewDatabase()
		instance.Spec.DatabaseName = "foo_dev"
		instance.Spec.Owner = "foo"

		// Set up secret for admin postgres connection
		passwordSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mysecret",
				Namespace: "summon-dev",
			},
			Data: map[string][]byte{
				"password": []byte("postgresadminconnpw"),
			},
		}
		ctx.Client = fake.NewFakeClient(instance, passwordSecret)
	})

	AfterEach(func() {
		db.Close()
		dbpool.Dbs.Delete("postgres host=mydb port=5432 dbname=foo_dev user=myuser password='postgresadminconnpw' sslmode=require")

		// Check for any unmet expectations.
		err := dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfulfilled database expectations: %s", err))
		}
	})

	Describe("IsReconcilable", func() {
		It("returns true when database and periscope postgresuser are ready", func() {
			comp = pdcomponents.NewPeriscopeUser()
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			instance.Status.SharedUsers.Periscope = dbv1beta1.StatusReady

			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

		It("returns false if PeriscopeUser was skipped", func() {
			comp = pdcomponents.NewPeriscopeUser()
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			instance.Status.SharedUsers.Periscope = dbv1beta1.StatusSkipped

			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})
	})

	It("grants the periscope user in db read-permissions", func() {
		comp = pdcomponents.NewPeriscopeUser()
		instance.Status.DatabaseStatus = dbv1beta1.StatusReady
		instance.Status.SharedUsers.Periscope = dbv1beta1.StatusReady

		// Reconciler checks that periscope user exists first.
		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		dbMock.ExpectQuery(`select COUNT\(\*\) from pg_user where usename='periscope'`).WillReturnRows(rows).RowsWillBeClosed()
		// Then checks periscope user for granted permissions. Here, we simulate no grants.
		rows = sqlmock.NewRows([]string{"count"}).AddRow(0)
		dbMock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema.table_privileges where grantee='periscope'`).WillReturnRows(rows).RowsWillBeClosed()
		// PeriscopeUserComponent should then grant permissions.
		dbMock.ExpectExec(`GRANT SELECT ON ALL TABLES IN SCHEMA public TO periscope`).WillReturnResult(sqlmock.NewResult(0, 1))
		// PeriscopeUserComponent should then alter default permissions.
		dbMock.ExpectExec(`ALTER DEFAULT PRIVILEGES FOR ROLE`).WillReturnResult(sqlmock.NewResult(0, 1))
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("does not grant the periscope user in db read-permissions if it already did before", func() {
		comp = pdcomponents.NewPeriscopeUser()
		instance.Status.DatabaseStatus = dbv1beta1.StatusReady
		instance.Status.SharedUsers.Periscope = dbv1beta1.StatusReady

		// Reconciler checks that periscope user exists first.
		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		dbMock.ExpectQuery(`select COUNT\(\*\) from pg_user where usename='periscope'`).WillReturnRows(rows).RowsWillBeClosed()
		// Then checks periscope user for granted permissions.
		rows = sqlmock.NewRows([]string{"count"}).AddRow(1)
		dbMock.ExpectQuery(`SELECT COUNT\(\*\) FROM information_schema.table_privileges where grantee='periscope'`).WillReturnRows(rows).RowsWillBeClosed()
		Expect(comp).To(ReconcileContext(ctx))
	})
})
