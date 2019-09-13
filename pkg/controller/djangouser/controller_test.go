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
	"net/url"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

const djangoSchema = `
CREATE TABLE auth_user (
    id SERIAL PRIMARY KEY,
    password character varying(128) NOT NULL,
    last_login timestamp with time zone,
    is_superuser boolean NOT NULL,
    username character varying(150) NOT NULL UNIQUE,
    first_name character varying(150) NOT NULL,
    last_name character varying(150) NOT NULL,
    email character varying(254) NOT NULL,
    is_staff boolean NOT NULL,
    is_active boolean NOT NULL,
    date_joined timestamp with time zone NOT NULL
);

CREATE TABLE common_userprofile (
    id SERIAL PRIMARY KEY,
    phone_number character varying(15),
    date_of_birth bytea,
    is_jumio_verified boolean NOT NULL,
    jumio_report_created_at timestamp with time zone,
    rfid character varying(21),
    rfid_to_be_updated character varying(21),
    pin_number character varying(32),
    mailing_address_display text,
    preferred_language character varying(32),
    driving_license_id integer,
    mailing_address_id integer,
    user_id integer NOT NULL UNIQUE REFERENCES auth_user (id),
    is_id_face_liveness_verified boolean,
    jwt_secret_key character varying(255),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    password_history character varying(128)[],
    external_membership_number character varying(255),
    jumio_scan_id character varying(36),
    jumio_uncorrected_data_submitted_at timestamp with time zone,
    uuid uuid,
    city text,
    postal_code text,
    state text,
    uber_user_id character varying(256)
);

CREATE TABLE common_staff (
    id SERIAL PRIMARY KEY,
    dispatcher boolean NOT NULL,
    manager boolean NOT NULL,
    is_active boolean NOT NULL,
    user_profile_id integer NOT NULL UNIQUE REFERENCES common_userprofile (id),
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);
`

var _ = Describe("DjangoUser controller @postgres", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *summonv1beta1.DjangoUser
	var adminConn, conn *dbv1beta1.PostgresConnection

	BeforeEach(func() {
		// Check for required environment variables.
		if os.Getenv("POSTGRES_URI") == "" {
			if os.Getenv("CI") == "" {
				Skip("Skipping Postgres controller tests")
			} else {
				Fail("Postgres test environment not configured")
			}
		}

		helpers = testHelpers.SetupTest()

		// Parse the Postgres database.
		parsed, err := url.Parse(os.Getenv("POSTGRES_URI"))
		if err != nil {
			Fail(err.Error())
		}
		adminConn = &dbv1beta1.PostgresConnection{
			Host:              parsed.Hostname(),
			Username:          parsed.User.Username(),
			PasswordSecretRef: apihelpers.SecretRef{Name: "pgpass"},
			Database:          parsed.Path[1:],
			SSLMode:           "disable",
		}
		conn = adminConn.DeepCopy()
		conn.Database, err = utils.RandomString(8)
		Expect(err).NotTo(HaveOccurred())

		// Set up the secret.
		password, _ := parsed.User.Password()
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "pgpass", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"password": []byte(password),
			},
		}
		helpers.TestClient.Create(dbSecret)

		// Scaffold a test instance.
		instance = &summonv1beta1.DjangoUser{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com", Namespace: helpers.Namespace},
			Spec: summonv1beta1.DjangoUserSpec{
				Active:    true,
				Staff:     true,
				Superuser: true,
				Database:  *conn,
			},
		}

		// Create the database.
		ctx := components.NewTestContext(instance, nil)
		ctx.Client = helpers.Client
		db, err := postgres.Open(ctx, adminConn)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec("CREATE DATABASE " + conn.Database)
		Expect(err).ToNot(HaveOccurred())

		// Import the schema.
		db, err = postgres.Open(ctx, conn)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(djangoSchema)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&summonv1beta1.DjangoUserList{})
		}

		helpers.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient

		// Create the DjangoUser.
		c.Create(instance)

		// Make sure it is created successfully.
		c.EventuallyGet(helpers.Name("foo.example.com"), instance, c.EventuallyStatus(summonv1beta1.StatusReady))
		Expect(instance.Status.Message).To(Equal("User 1 created"))
	})
})
