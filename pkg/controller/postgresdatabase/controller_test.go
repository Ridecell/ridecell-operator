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

package postgresdatabase_test

import (
	"fmt"
	"net/url"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_sql"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

const timeout = time.Second * 20

var _ = Describe("PostgresDatabase controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var randomName string
	var instance *dbv1beta1.PostgresDatabase
	var conn *dbv1beta1.PostgresConnection
	var dbconfig *dbv1beta1.DbConfig

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
		conn = &dbv1beta1.PostgresConnection{
			Host:              parsed.Hostname(),
			Username:          parsed.User.Username(),
			PasswordSecretRef: apihelpers.SecretRef{Name: "pgpass"},
			Database:          parsed.Path[1:],
			SSLMode:           "disable",
		}
		password, _ := parsed.User.Password()
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "pgpass", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"password": []byte(password),
			},
		}
		helpers.TestClient.Create(secret)

		randomName = utils.RandomString(4)
		instance = &dbv1beta1.PostgresDatabase{
			ObjectMeta: metav1.ObjectMeta{Name: randomName + "-dev", Namespace: helpers.Namespace},
		}
		dbconfig = &dbv1beta1.DbConfig{
			ObjectMeta: metav1.ObjectMeta{Name: helpers.Namespace, Namespace: helpers.Namespace},
		}
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&dbv1beta1.PostgresDatabaseList{})
			helpers.DebugList(&dbv1beta1.RDSInstanceList{})
			helpers.DebugList(&dbv1beta1.PostgresUserList{})
			helpers.DebugList(&dbv1beta1.DbConfigList{})
		}

		helpers.TeardownTest()
	})

	It("creates a database on an exclusive RDS config", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Spec.Postgres.Mode = "Exclusive"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Create our database.
		c.Create(instance)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(randomName+"-dev"), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))

		// Try to connect.
		ctx := components.NewTestContext(instance, nil)
		ctx.Client = helpers.Client
		db, err := postgres.Open(ctx, &instance.Status.Connection)
		Expect(err).ToNot(HaveOccurred())

		// Make a table.
		_, err = db.Exec(`CREATE TABLE testing (id SERIAL, str VARCHAR)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, randomName)
		Expect(err).ToNot(HaveOccurred())
		row := db.QueryRow(`SELECT id FROM testing WHERE str = $1`, randomName)
		var rowId int
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))
	})

	It("creates a database on a shared RDS config", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Spec.Postgres.Mode = "Shared"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Create our database.
		c.Create(instance)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(helpers.Namespace), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))

		// Try to connect.
		ctx := components.NewTestContext(instance, nil)
		ctx.Client = helpers.Client
		db, err := postgres.Open(ctx, &instance.Status.Connection)
		Expect(err).ToNot(HaveOccurred())

		// Make a table.
		_, err = db.Exec(`CREATE TABLE testing (id SERIAL, str VARCHAR)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, randomName)
		Expect(err).ToNot(HaveOccurred())
		row := db.QueryRow(`SELECT id FROM testing WHERE str = $1`, randomName)
		var rowId int
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))
	})

	It("creates a database on an exclusive Local config", func() {
		c := helpers.TestClient

		// Inject a mock database.
		dbpool.Dbs.Store(fmt.Sprintf("postgres host=%s-dev-database port=5432 dbname=postgres user=ridecell-admin password='password' sslmode=require", randomName), fake_sql.Open())
		dbpool.Dbs.Store(fmt.Sprintf("postgres host=%s-dev-database port=5432 dbname=postgres user=%s_dev password='userpassword' sslmode=require", randomName, randomName), fake_sql.Open())

		// Fudge the postgres user password so it isn't a random value.
		userSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-dev.postgres-user-password", randomName), Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"password": []byte("userpassword"),
			},
		}
		c.Create(userSecret)

		// Set up the DbConfig.
		dbconfig.Spec.Postgres.Mode = "Exclusive"
		dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
		c.Create(dbconfig)

		// Create our database.
		c.Create(instance)

		// Get our Local cluster and advance it to ready.
		pg := &postgresv1.Postgresql{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-database"), pg)
		pg.Status = postgresv1.ClusterStatusRunning
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ridecell-admin.%s-dev-database.credentials", randomName), Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"password": []byte("password"),
			},
		}
		c.Create(dbSecret)
		c.Status().Update(pg)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))
	})

	It("supports cross namespace use for shared mode", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Name = randomName
		dbconfig.Namespace = helpers.OperatorNamespace
		dbconfig.Spec.Postgres.Mode = "Shared"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Create our database.
		instance.Spec.DbConfigRef.Name = randomName
		instance.Spec.DbConfigRef.Namespace = helpers.OperatorNamespace
		c.Create(instance)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(types.NamespacedName{Name: randomName, Namespace: helpers.OperatorNamespace}, rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))

		// Try to connect.
		ctx := components.NewTestContext(instance, nil)
		ctx.Client = helpers.Client
		db, err := postgres.Open(ctx, &instance.Status.Connection)
		Expect(err).ToNot(HaveOccurred())

		// Make a table.
		_, err = db.Exec(`CREATE TABLE testing (id SERIAL, str VARCHAR)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, randomName)
		Expect(err).ToNot(HaveOccurred())
		row := db.QueryRow(`SELECT id FROM testing WHERE str = $1`, randomName)
		var rowId int
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))
	})

	It("supports cross namespace use for exclusive mode", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Name = randomName
		dbconfig.Namespace = helpers.OperatorNamespace
		dbconfig.Spec.Postgres.Mode = "Exclusive"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Create our database.
		instance.Spec.DbConfigRef.Name = randomName
		instance.Spec.DbConfigRef.Namespace = helpers.OperatorNamespace
		c.Create(instance)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(randomName+"-dev"), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))

		// Try to connect.
		ctx := components.NewTestContext(instance, nil)
		ctx.Client = helpers.Client
		db, err := postgres.Open(ctx, &instance.Status.Connection)
		Expect(err).ToNot(HaveOccurred())

		// Make a table.
		_, err = db.Exec(`CREATE TABLE testing (id SERIAL, str VARCHAR)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, randomName)
		Expect(err).ToNot(HaveOccurred())
		row := db.QueryRow(`SELECT id FROM testing WHERE str = $1`, randomName)
		var rowId int
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))
	})
})
