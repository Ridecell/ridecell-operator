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

package postgresuser_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_sql"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("postgresuser controller", func() {
	var perTestHelper *test_helpers.PerTestHelpers
	var instance dbv1beta1.PostgresUser

	BeforeEach(func() {
		perTestHelper = testHelpers.SetupTest()
		c := testHelpers.TestClient
		// password for operator user
		passwordSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "password-secret",
				Namespace: perTestHelper.Namespace,
			},
			Data: map[string][]byte{
				"password": []byte("password"),
			},
		}
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test.postgres-user-password",
				Namespace: perTestHelper.Namespace,
			},
			Data: map[string][]byte{
				"password": []byte("newpassword"),
			},
		}

		c.Create(passwordSecret)
		c.Create(newSecret)

		instance = dbv1beta1.PostgresUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: perTestHelper.Namespace,
			},
			Spec: dbv1beta1.PostgresUserSpec{
				Connection: dbv1beta1.PostgresConnection{
					Host:     "test-database",
					Port:     5432,
					Username: "operator",
					PasswordSecretRef: helpers.SecretRef{
						Name: "password-secret",
						Key:  "password",
					},
					Database: "test",
				},
				Username: "newuser",
			},
			Status: dbv1beta1.PostgresUserStatus{
				PasswordSecretRef: helpers.SecretRef{
					Name: "test.postgres-user-password",
					Key:  "password",
				},
			},
		}
		dbpool.Dbs.Store("postgres host=test-database port=5432 dbname=test user=operator password='password' sslmode=require", fake_sql.Open())
		dbpool.Dbs.Store("postgres host=test-database port=5432 dbname=postgres user=newuser password='newpassword' sslmode=require", fake_sql.Open())
	})

	AfterEach(func() {
		perTestHelper.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := testHelpers.TestClient
		c.Create(&instance)

		fetchInstance := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(perTestHelper.Name("test"), fetchInstance, c.EventuallyStatus(dbv1beta1.StatusReady))

		Expect(fetchInstance.Status.SecretStatus).To(Equal(dbv1beta1.StatusReady))
	})
})
