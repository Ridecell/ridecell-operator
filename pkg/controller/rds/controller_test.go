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

package rds_test

import (
	"os"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/sts"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sess *session.Session
var rdssvc *rds.RDS
var rdsInstance *dbv1beta1.RDSInstance
var randOwnerPrefix string

var _ = Describe("rds controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()

		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping rds integration tests")
		}

		if os.Getenv("AWS_SUBNET_GROUP_NAME") == "" {
			panic("$AWS_SUBNET_GROUP_NAME not set, failing test")
		}

		if os.Getenv("VPC_ID") == "" {
			panic("$VPC_ID not set, failing test")
		}

		var err error
		sess, err = session.NewSession(&aws.Config{
			Region: aws.String("us-west-1"),
		})
		Expect(err).NotTo(HaveOccurred())

		// Check if this being run on the testing account
		stssvc := sts.New(sess)
		getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		Expect(err).NotTo(HaveOccurred())
		if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
			panic("These tests should only be run on the testing account.")
		}

		rdssvc = rds.New(sess)

		rdsInstance = &dbv1beta1.RDSInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
		}
		rdsInstance.Spec.Username = "testUsername"
		// multiaz false should save some time in testing while being operationally similar to normal usage
		rdsInstance.Spec.MultiAZ = aws.Bool(false)
	})

	AfterEach(func() {
		// Delete object and see if it cleans up on its own
		//c := helpers.TestClient

		//c.Delete(rdsInstance)
		//Eventually(func() error { return bucketExists() }, time.Second*10).ShouldNot(Succeed())

		helpers.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient
		c.Create(rdsInstance)

		fetchRDS := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name("test"), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusCreating))

		// Initial testing pegs full db creation at 12 with multiaz minutes, lets use 15 to be safe.
		c.EventuallyGet(helpers.Name("test"), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusReady), c.EventuallyTimeout(time.Minute*15))

		fetchSecret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("test.rds.credentials"), fetchSecret)
		Expect(string(fetchSecret.Data["username"])).To(Equal("testUsername"))
		Expect(len(string(fetchSecret.Data["password"]))).To(BeNumerically(">", 0))
		Expect(len(string(fetchSecret.Data["endpoint"]))).To(BeNumerically(">", 0))

		// We need a component context to user the helpers here
		testContext := components.NewTestContext(fetchRDS, nil)
		// Make sure we can query the database, while we're at it make sure our user is in the rds_superuser group
		db, err := postgres.Open(testContext, &rdsInstance.Status.RDSConnection)
		defer db.Close()
		Expect(err).ToNot(HaveOccurred())

		output, err := db.Query(`SELECT groname FROM pg_group WHERE (SELECT usesysid FROM pg_user WHERE usename = current_user)=ANY(grolist);`)
		defer output.Close()
		Expect(err).ToNot(HaveOccurred())

		var column0 string
		for output.Next() {
			err := output.Scan(&column0)
			Expect(err).ToNot(HaveOccurred())
			Expect(column0).To(Equal("rds_superuser"))
		}

		// Make sure that the database will update password if it is lost somehow
		c.Delete(fetchSecret)
		c.EventuallyGet(helpers.Name("test"), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusModifying))
		// It also should go back to ready state
		c.EventuallyGet(helpers.Name("test"), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusReady))

		c.EventuallyGet(helpers.Name("test.rds.credentials"), fetchSecret)
		Expect(string(fetchSecret.Data["username"])).To(Equal("testUsername"))
		Expect(len(string(fetchSecret.Data["password"]))).To(BeNumerically(">", 0))
		Expect(len(string(fetchSecret.Data["endpoint"]))).To(BeNumerically(">", 0))

		db, err = postgres.Open(testContext, &rdsInstance.Status.RDSConnection)
		defer db.Close()
		Expect(err).ToNot(HaveOccurred())

		// run a test query, bonus make sure we're rds_superuser
		output, err = db.Query(`SELECT groname FROM pg_group WHERE (SELECT usesysid FROM pg_user WHERE usename = current_user)=ANY(grolist);`)
		defer output.Close()
		Expect(err).ToNot(HaveOccurred())

		for output.Next() {
			err := output.Scan(&column0)
			Expect(err).ToNot(HaveOccurred())
			Expect(column0).To(Equal("rds_superuser"))
		}

		// Lets edit the parameter group!
		rdsInstance.Spec.Parameters = map[string]string{
			"log_min_duration_statement": "5000",
		}
		//c.Update(rdsInstance)
		//c.EventuallyGet(helpers.Name("test"), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusModifying))
		// In this case the parameter change should be applied immediately, others will put the db into pending-reboot
		//c.EventuallyGet(helpers.Name("test"), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusReady))

	})

})
