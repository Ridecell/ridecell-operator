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
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	djangousercomponents "github.com/Ridecell/ridecell-operator/pkg/controller/djangouser/components"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

type passwordMatching struct {
	password string
}

// This isn't a great test because it's really just checking GIGO, but it's the
// best I can do in a unit test.
func (p passwordMatching) Match(v driver.Value) bool {
	hashAndData, ok := v.(string)
	if !ok {
		return false
	}
	hash := strings.SplitN(hashAndData, "$", 2)[1]
	digested := sha256.Sum256([]byte(p.password))
	encoded := hex.EncodeToString(digested[:])
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(encoded))
	return err == nil
}

// Short test to test the test matcher used later on.
var _ = Describe("passwordMatching", func() {
	// Created by `manage.py changepassword
	djangoHash := "bcrypt_sha256$$2b$12$c6T/4vpvrWs5NhmfIst8O.od24KswgzMu/9Uf1UXHKt2tHuit118i"

	badHash := "bcrypt_sha256$$2b$12$ONmUvu/6O3xmMqutasdf3OXLJsePlLOs5vxN4YnGsCRlPxCR.CCa."

	It("accepts a hash created by Django", func() {
		m := passwordMatching{password: "secretdbpass"}
		Expect(m.Match(djangoHash)).To(BeTrue())
	})

	It("rejects various bad hashes", func() {
		m := passwordMatching{password: "secretdbpass"}
		Expect(m.Match(badHash)).To(BeFalse())
		m = passwordMatching{password: "other"}
		Expect(m.Match(djangoHash)).To(BeFalse())
	})
})

var _ = Describe("DjangoUser Database Component", func() {
	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		var err error
		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=foo-database port=5432 dbname=summon user=summon password='secretdbpass' sslmode=require", db)

		// Baseline test instance.
		instance.Spec = summonv1beta1.DjangoUserSpec{
			Email: "foo@example.com",
			Database: dbv1beta1.PostgresConnection{
				Host:     "foo-database",
				Port:     5432,
				Username: "summon",
				Database: "summon",
				PasswordSecretRef: helpers.SecretRef{
					Name: "summon.foo-database.credentials",
					Key:  "password",
				},
			},
		}
	})

	AfterEach(func() {
		db.Close()
		dbpool.Dbs.Delete("postgres host=foo-database port=5432 dbname=summon user=summon password='secretdbpass' sslmode=require")

		// Check for any unmet expectations.
		err := dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfulfilled database expectations: %s", err))
		}
	})

	It("creates a user", func() {
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.foo-database.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("secretdbpass"),
			},
		}
		userSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com.django-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("djangopass"),
			},
		}
		ctx.Client = fake.NewFakeClient(dbSecret, userSecret)

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		dbMock.ExpectQuery("INSERT INTO auth_user").WithArgs("foo@example.com", passwordMatching{password: "djangopass"}, "", "", false, false, false).WillReturnRows(rows)
		dbMock.ExpectQuery("INSERT INTO common_userprofile").WithArgs(1).WillReturnRows(rows)
		dbMock.ExpectExec("INSERT INTO common_staff").WithArgs(1, false, false, false).WillReturnResult(sqlmock.NewResult(0, 1))

		comp := djangousercomponents.NewDatabase()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusReady))
		Expect(instance.Status.Message).To(Equal("User 1 created"))
	})

	It("obeys the status flags", func() {
		instance.Spec.Active = true
		instance.Spec.Staff = true
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.foo-database.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("secretdbpass"),
			},
		}
		userSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com.django-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("djangopass"),
			},
		}
		ctx.Client = fake.NewFakeClient(dbSecret, userSecret)

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		dbMock.ExpectQuery("INSERT INTO auth_user").WithArgs("foo@example.com", passwordMatching{password: "djangopass"}, "", "", true, true, false).WillReturnRows(rows)
		dbMock.ExpectQuery("INSERT INTO common_userprofile").WithArgs(1).WillReturnRows(rows)
		dbMock.ExpectExec("INSERT INTO common_staff").WithArgs(1, true, false, false).WillReturnResult(sqlmock.NewResult(0, 1))

		comp := djangousercomponents.NewDatabase()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusReady))
		Expect(instance.Status.Message).To(Equal("User 1 created"))
	})

	It("sets the first and last name", func() {
		instance.Spec.FirstName = "Alan"
		instance.Spec.LastName = "Smithee"
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.foo-database.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("secretdbpass"),
			},
		}
		userSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com.django-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("djangopass"),
			},
		}
		ctx.Client = fake.NewFakeClient(dbSecret, userSecret)

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		dbMock.ExpectQuery("INSERT INTO auth_user").WithArgs("foo@example.com", passwordMatching{password: "djangopass"}, "Alan", "Smithee", false, false, false).WillReturnRows(rows)
		dbMock.ExpectQuery("INSERT INTO common_userprofile").WithArgs(1).WillReturnRows(rows)
		dbMock.ExpectExec("INSERT INTO common_staff").WithArgs(1, false, false, false).WillReturnResult(sqlmock.NewResult(0, 1))

		comp := djangousercomponents.NewDatabase()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusReady))
		Expect(instance.Status.Message).To(Equal("User 1 created"))
	})

	It("errors if the auth_user insert fails", func() {
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.foo-database.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("secretdbpass"),
			},
		}
		userSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com.django-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("djangopass"),
			},
		}
		ctx.Client = fake.NewFakeClient(dbSecret, userSecret)

		dbMock.ExpectQuery("INSERT INTO auth_user").WillReturnError(fmt.Errorf("Table auth_user is on fire"))

		comp := djangousercomponents.NewDatabase()
		Expect(comp).ToNot(ReconcileContext(ctx))
	})
})
