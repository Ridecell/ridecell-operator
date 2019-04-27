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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	pdcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresdatabase/components"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("PostgresDatabase Database Component", func() {
	var comp components.Component
	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=mydb port=5432 dbname=postgres user=myuser password='mypassword' sslmode=require", db)

		comp = pdcomponents.NewDatabase()
		instance.Spec.DatabaseName = "foo_dev"
		instance.Spec.Owner = "foo"
		instance.Status.AdminConnection = dbv1beta1.PostgresConnection{
			Host:     "mydb",
			Port:     5432,
			Username: "myuser",
			PasswordSecretRef: helpers.SecretRef{
				Name: "mysecret",
				Key:  "password",
			},
			Database: "postgres",
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"password": []byte("mypassword"),
			},
		}
		ctx.Client = fake.NewFakeClient(instance, secret)
	})

	Describe("IsReconcilable", func() {
		It("does create a database if the database and user are ready", func() {
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			instance.Status.UserStatus = dbv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

		It("does create a database if the database is ready and skipuser", func() {
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			instance.Spec.SkipUser = true
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

		It("does not create a database if the database is not ready", func() {
			instance.Status.DatabaseStatus = ""
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("does not create a database if the iser is not ready", func() {
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			instance.Status.UserStatus = ""
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})
	})

	It("creates a database", func() {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		dbMock.ExpectQuery(`SELECT COUNT`).WithArgs("foo_dev").WillReturnRows(rows)
		dbMock.ExpectExec(`CREATE DATABASE "foo_dev" WITH OWNER = 'foo'`).WillReturnResult(sqlmock.NewResult(0, 1))

		Expect(comp).To(ReconcileContext(ctx))

		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
	})
})
