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

package djangouser_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

const timeout = time.Second * 20

var _ = Describe("Summon controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()

		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=foo-database port=5432 dbname=summon user=summon password='secretdbpass' sslmode=require", db)
	})

	AfterEach(func() {
		helpers.TeardownTest()
		db.Close()
		dbpool.Dbs.Delete("postgres host=foo-database port=5432 dbname=summon user=summon password='secretdbpass' sslmode=require")

		// Check for any unmet expectations.
		err := dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfulfilled database expectations: %s", err))
		}
	})

	It("runs a basic reconcile", func() {
		instance := &summonv1beta1.DjangoUser{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com", Namespace: helpers.Namespace},
			Spec: summonv1beta1.DjangoUserSpec{
				Active:    true,
				Staff:     true,
				Superuser: true,
				Database: dbv1beta1.PostgresConnection{
					Host:     "foo-database",
					Username: "summon",
					Database: "summon",
					PasswordSecretRef: apihelpers.SecretRef{
						Name: "summon.foo-database.credentials",
					},
				},
			},
		}
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.foo-database.credentials", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"password": []byte("secretdbpass"),
			},
		}

		rows := sqlmock.NewRows([]string{"id"}).AddRow(123)
		dbMock.ExpectQuery("INSERT INTO auth_user").WillReturnRows(rows).AnyNumberOfTimes()
		dbMock.ExpectQuery("INSERT INTO common_userprofile").WillReturnRows(rows).AnyNumberOfTimes()
		dbMock.ExpectExec("INSERT INTO common_staff").WillReturnResult(sqlmock.NewResult(0, 1)).AnyNumberOfTimes()

		err := helpers.Client.Create(context.TODO(), dbSecret)
		Expect(err).NotTo(HaveOccurred())

		err = helpers.Client.Create(context.TODO(), instance)
		Expect(err).NotTo(HaveOccurred())

		fetched := &summonv1beta1.DjangoUser{}
		Eventually(func() (string, error) {
			err := helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo.example.com", Namespace: helpers.Namespace}, fetched)
			if err != nil {
				return "", err
			}
			return fetched.Status.Status, nil
		}, timeout).Should(Equal(summonv1beta1.StatusReady))
		Expect(fetched.Status.Message).To(Equal("User 123 created"))
	})
})
