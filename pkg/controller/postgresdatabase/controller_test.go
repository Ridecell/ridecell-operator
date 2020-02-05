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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

var _ = Describe("PostgresDatabase controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var randomName string
	var instance *dbv1beta1.PostgresDatabase
	var conn *dbv1beta1.PostgresConnection
	var dbconfig *dbv1beta1.DbConfig

	validateUserReadOnlyPermissions := func(ctx *components.ComponentContext, user *dbv1beta1.PostgresUser, pgdb *dbv1beta1.PostgresDatabase, dbName string) {
		// Connect as given user into the desired database.
		userconn := user.Status.Connection.DeepCopy()
		userconn.Database = pgdb.Status.Connection.Database

		db, err := postgres.Open(ctx, userconn)
		Expect(err).ToNot(HaveOccurred())

		// User should not be able to insert values!
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, dbName)
		Expect(err).To(HaveOccurred())

		// User should be able to read from table.
		row := db.QueryRow(`SELECT id FROM testing WHERE str = $1`, dbName)
		var rowId int
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))
	}

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

		randomName, err = utils.RandomString(4)
		Expect(err).NotTo(HaveOccurred())
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

		instance.Spec.DbConfigRef.Namespace = helpers.Namespace

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

		// Expect periscope postgresuser to be created.
		puser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-periscope"), puser, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Expect reporting postgresuser to be created.
		ruser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-periscope"), ruser, c.EventuallyStatus(dbv1beta1.StatusReady))

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

		// Validate periscope user has read-only permissions
		validateUserReadOnlyPermissions(ctx, puser, instance, randomName)
		
		// Validate reporting user has read-only permissions
		validateUserReadOnlyPermissions(ctx, ruser, instance, randomName)
	})

	It("creates a database on a shared RDS config", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Spec.Postgres.Mode = "Shared"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(helpers.Namespace), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Create our database.
		instance.Spec.DbConfigRef.Namespace = helpers.Namespace
		c.Create(instance)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Expect periscope postgresuser to be created. Since the RDS is shared, only one periscope user object
		// created, and it's name follows the <namespace>-periscope instead of <database>-dev-periscope.
		puser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(helpers.Namespace+"-periscope"), puser, c.EventuallyStatus(dbv1beta1.StatusReady))

		ruser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(helpers.Namespace+"-reporting"), ruser, c.EventuallyStatus(dbv1beta1.StatusReady))
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

		// Validate periscope user has read-only permissions
		validateUserReadOnlyPermissions(ctx, puser, instance, randomName)

		// Validate reporting user has read-only permissions
		validateUserReadOnlyPermissions(ctx, ruser, instance, randomName)
	})

	// Since this is a mock local db, can't really test dbuser connection, but might as well test that NoCreatePeriscopeUser 
	// and NoCreateReportingUser logic works.
	It("creates a database on an exclusive Local config without Periscope and Reporting (NoCreate*User flag)", func() {
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
		dbconfig.Spec.NoCreatePeriscopeUser = true
		dbconfig.Spec.NoCreateReportingUser = true
		c.Create(dbconfig)

		// Create our database.
		instance.Spec.DbConfigRef.Namespace = helpers.Namespace
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

		// Wait for our database to become ready. Expecting Skip status.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyValue(Equal("Skipped"), func(obj runtime.Object) (interface{}, error) {
			return obj.(*dbv1beta1.PostgresDatabase).Status.SharedUsers.Periscope, nil
		}))
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))

		// Check that no periscope or reporting user exists and SharedUser Status for PostgresDatabase is skipped.
		ctx := components.NewTestContext(instance, nil)
		user := &dbv1beta1.PostgresUser{}
		err := ctx.Get(ctx.Context, helpers.Name(randomName+"-periscope"), user)
		Expect(err).To(HaveOccurred())
		err = ctx.Get(ctx.Context, helpers.Name(randomName+"-reporting"), user)
		Expect(err).To(HaveOccurred())
		Expect(instance.Status.SharedUsers.Periscope).To(Equal("Skipped"))
		Expect(instance.Status.SharedUsers.Reporting).To(Equal("Skipped"))
		// When dbconfig mode is exclusive, we don't update it's postgres status, so not checking dbconfig's periscope status.
		
	})

	// This test case creates a PostgresDatabase object in helpers.Namespace, but references a DbConfig (and its corresponding RDS instance)
	// from helpers.OperatorNamespace. Periscope postgresuser gets created under DbConfig's namespace, rather than PostgresDatabase's namespace.
	// Later, when we connect to the database as the periscope postgresuser, the retrieval of periscope postgresuser's secret is attempted under
	// the current namespace. However, the secret only exists under the DbConfig namespace, so the secret retrieval will fail. As a workaround,
	// we create a copy of periscope postgresuser secret in the current namespace.
	// This test is flakey due to some secret failing to be fetched (in time?)
	It("supports cross namespace use for shared mode", func() {
		c := helpers.TestClient

		// Workaround: copy postgreuser secrets in current namespace
		periscope_secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: randomName + "-periscope.postgres-user-password", Namespace: instance.Namespace},
			Data: map[string][]byte{
				"password": []byte("foo"),
			},
		}
		c.Create(periscope_secret)

		reporting_secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: randomName + "-reporting.postgres-user-password", Namespace: instance.Namespace},
			Data: map[string][]byte{
				"password": []byte("foo"),
			},
		}
		c.Create(reporting_secret)

		// Create user secrets under dbconfig namespace
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "pgpass-crossnamespace", Namespace: helpers.OperatorNamespace},
			Data: map[string][]byte{
				"password": []byte("test"),
			},
		}

		c.Create(newSecret)

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
		conn.PasswordSecretRef = apihelpers.SecretRef{Name: "pgpass-crossnamespace"}
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Confirm our secret is copied over
		fetchSecret := &corev1.Secret{}
		c.EventuallyGet(types.NamespacedName{Name: "pgpass-crossnamespace", Namespace: helpers.Namespace}, fetchSecret)

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))

		// Expect periscope postgresuser to be created under DBConfig's namespace, since postgresdb instance
		// is referencing a Shared DBConfig and has a RDS instance.
		puser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(types.NamespacedName{Name: randomName + "-periscope", Namespace: helpers.OperatorNamespace}, puser)

		ruser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(types.NamespacedName{Name: randomName + "-reporting", Namespace: helpers.OperatorNamespace}, ruser)

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

		// Validate periscope user has read-only permissions
		validateUserReadOnlyPermissions(ctx, puser, instance, randomName)

		// Validate reporting user has read-only permissions
		validateUserReadOnlyPermissions(ctx, ruser, instance, randomName)
	})

	It("supports cross namespace use for exclusive mode", func() {
		c := helpers.TestClient

		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "pgpass-samenamespace", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"password": []byte("test"),
			},
		}

		c.Create(newSecret)

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
		conn.PasswordSecretRef = apihelpers.SecretRef{Name: "pgpass-samenamespace"}
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Confirm our secret is copied over
		fetchSecret := &corev1.Secret{}
		c.EventuallyGet(types.NamespacedName{Name: "pgpass-samenamespace", Namespace: helpers.Namespace}, fetchSecret)

		// Expect periscope postgresuser to be created.
		puser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-periscope"), puser, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Expect reporting postgresuser to be created.
		ruser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-periscope"), ruser, c.EventuallyStatus(dbv1beta1.StatusReady))

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

		// Validate periscope user has read-only permissions
		validateUserReadOnlyPermissions(ctx, puser, instance, randomName)

		// Validate reporting user has read-only permissions
		validateUserReadOnlyPermissions(ctx, ruser, instance, randomName)
	})

	It("creates two databases on a shared RDS config, but only one periscope and one reporting postgresuser object", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Spec.Postgres.Mode = "Shared"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Secondary database
		instance2 := &dbv1beta1.PostgresDatabase{
			ObjectMeta: metav1.ObjectMeta{Name: randomName + "-dev2", Namespace: helpers.Namespace},
		}

		// Create our databases.
		c.Create(instance)
		c.Create(instance2)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(helpers.Namespace), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our database to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))
		c.EventuallyGet(helpers.Name(randomName+"-dev2"), instance2, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check the output connection.
		Expect(instance.Status.Connection.Database).ToNot(Equal("postgres"))
		Expect(instance2.Status.Connection.Database).ToNot(Equal("postgres"))

		// Expect a single periscope postgresuser to be created since we are using Shared DbConfig.
		puser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(helpers.Namespace+"-periscope"), puser, c.EventuallyStatus(dbv1beta1.StatusReady))

		ruser := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(helpers.Namespace+"-reporting"), ruser, c.EventuallyStatus(dbv1beta1.StatusReady))

		ctx := components.NewTestContext(instance, nil)
		err := ctx.Get(ctx.Context, helpers.Name(randomName+"-dev-periscope"), puser)
		Expect(err).To(HaveOccurred())
		err = ctx.Get(ctx.Context, helpers.Name(randomName+"-dev-reporting"), ruser)
		Expect(err).To(HaveOccurred())

		// Try to connect.
		ctx = components.NewTestContext(instance, nil)
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

		// Validate periscope user has read-only permissions to instance1
		validateUserReadOnlyPermissions(ctx, puser, instance, randomName)

		// Validate reporting user has read-only permissions to instance1
		validateUserReadOnlyPermissions(ctx, ruser, instance, randomName)
		
		// Repeat for instance2.
		ctx = components.NewTestContext(instance2, nil)
		err = ctx.Get(ctx.Context, helpers.Name(randomName+"-dev2-periscope"), puser)
		Expect(err).To(HaveOccurred())

		// Try to connect.
		ctx = components.NewTestContext(instance2, nil)
		ctx.Client = helpers.Client
		db, err = postgres.Open(ctx, &instance2.Status.Connection)
		Expect(err).ToNot(HaveOccurred())

		// Make a table.
		_, err = db.Exec(`CREATE TABLE testing (id SERIAL, str VARCHAR)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, randomName+"2")
		Expect(err).ToNot(HaveOccurred())
		row = db.QueryRow(`SELECT id FROM testing WHERE str = $1`, randomName+"2")
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))

		// Validate periscope user has read-only permissions to instance2
		validateUserReadOnlyPermissions(ctx, puser, instance2, randomName+"2")

		// Validate reporting user has read-only permissions to instance2
		validateUserReadOnlyPermissions(ctx, ruser, instance2, randomName+"2")
	})

	It("creates two database on an exclusive RDS config, each having its own periscope and reporting postgresuser object", func() {
		c := helpers.TestClient

		// Set up the DbConfig.
		dbconfig.Spec.Postgres.Mode = "Exclusive"
		dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		c.Create(dbconfig)

		// Secondary database.
		instance2 := &dbv1beta1.PostgresDatabase{
			ObjectMeta: metav1.ObjectMeta{Name: randomName + "-dev2", Namespace: helpers.Namespace},
		}

		// Create our databases.
		c.Create(instance)
		c.Create(instance2)

		// Get our RDS cluster and advance it to ready.
		rds := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(randomName+"-dev"), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Get our second RDS cluster and advance it to ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev2"), rds)
		rds.Status.Status = dbv1beta1.StatusReady
		rds.Status.Connection = *conn
		c.Status().Update(rds)

		// Wait for our databases to become ready.
		c.EventuallyGet(helpers.Name(randomName+"-dev"), instance, c.EventuallyStatus(dbv1beta1.StatusReady))
		c.EventuallyGet(helpers.Name(randomName+"-dev2"), instance2, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Expect two periscope postgresusers to be created.
		puser := &dbv1beta1.PostgresUser{}
		puser2 := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-periscope"), puser, c.EventuallyStatus(dbv1beta1.StatusReady))
		c.EventuallyGet(helpers.Name(randomName+"-dev2-periscope"), puser2, c.EventuallyStatus(dbv1beta1.StatusReady))
		Expect(puser).ToNot(Equal(puser2))

		// Expect two reporting postgresusers to be created.
		ruser := &dbv1beta1.PostgresUser{}
		ruser2 := &dbv1beta1.PostgresUser{}
		c.EventuallyGet(helpers.Name(randomName+"-dev-reporting"), ruser, c.EventuallyStatus(dbv1beta1.StatusReady))
		c.EventuallyGet(helpers.Name(randomName+"-dev2-reporting"), ruser2, c.EventuallyStatus(dbv1beta1.StatusReady))
		Expect(puser).ToNot(Equal(ruser2))

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

		// Validate periscope user has read-only permissions
		validateUserReadOnlyPermissions(ctx, puser, instance, randomName)

		// Validate reporting user has read-only permissions
		validateUserReadOnlyPermissions(ctx, ruser, instance, randomName)

		// Repeat for instance2.
		// Try to connect.
		ctx = components.NewTestContext(instance2, nil)
		ctx.Client = helpers.Client
		db, err = postgres.Open(ctx, &instance2.Status.Connection)
		Expect(err).ToNot(HaveOccurred())

		// Make a table.
		_, err = db.Exec(`CREATE TABLE testing (id SERIAL, str VARCHAR)`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO testing (str) VALUES ($1)`, randomName+"2")
		Expect(err).ToNot(HaveOccurred())
		row = db.QueryRow(`SELECT id FROM testing WHERE str = $1`, randomName+"2")
		err = row.Scan(&rowId)
		Expect(err).ToNot(HaveOccurred())
		Expect(rowId).To(Equal(1))

		// Validate periscope user2 has read-only permissions to instance2
		validateUserReadOnlyPermissions(ctx, puser2, instance2, randomName+"2")

		// Validate reporting user2 has read-only permissions to instance2
		validateUserReadOnlyPermissions(ctx, ruser2, instance2, randomName+"2")

		// puser and ruser should only be able to access instance1 db (since it has its own RDS)
		// and puser2 and ruser2 should only be able access instance2 db. Not testing since
		// test environment only capable of hosting one database cluster (RDS), not two, to actually
		// test there's no mix up of user access and permissions.
	})
})
