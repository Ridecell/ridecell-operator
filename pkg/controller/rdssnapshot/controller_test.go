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

package rdssnapshot_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var rdsSnapshot *dbv1beta1.RDSSnapshot
var testHelpers *test_helpers.TestHelpers
var rdssvc *rds.RDS
var rdsInstanceID *string

var _ = Describe("rds controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping rdssnapshot integration tests")
		}

		if os.Getenv("AWS_SUBNET_GROUP_NAME") == "" {
			panic("$AWS_SUBNET_GROUP_NAME not set, failing test")
		}

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-west-1"),
		})
		Expect(err).ToNot(HaveOccurred())

		// Check if this being run on the testing account
		stssvc := sts.New(sess)
		getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		Expect(err).NotTo(HaveOccurred())
		if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
			panic("These tests should only be run on the testing account.")
		}

		rdssvc = rds.New(sess)

		randOwnerPrefix := os.Getenv("RAND_OWNER_PREFIX")
		if randOwnerPrefix == "" {
			panic("$RAND_OWNER_PREFIX not set, failing test")
		}

		rdsInstanceName := fmt.Sprintf("%s-snapshot-controller", randOwnerPrefix)
		rdsInstanceID, err = setupRDSInstance(rdsInstanceName)
		Expect(err).ToNot(HaveOccurred())
		helpers = testHelpers.SetupTest()

		rdsSnapshot = &dbv1beta1.RDSSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "snapshot-controller",
			},
			Spec: dbv1beta1.RDSSnapshotSpec{
				RDSInstanceID: *rdsInstanceID,
			},
		}
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&dbv1beta1.RDSSnapshotList{})
		}
	})

	It("creates a snapshot with no ttl", func() {
		c := helpers.TestClient
		rdsSnapshot.Name =
			c.Create(rdsSnapshot)

		fetchSnapshot := &dbv1beta1.RDSSnapshot{}
		c.EventuallyGet(helpers.Name(rdsSnapshot.Name), fetchSnapshot, c.EventuallyStatus(dbv1beta1.StatusCreating))
		c.EventuallyGet(helpers.Name(rdsSnapshot.Name), fetchSnapshot, c.EventuallyStatus(dbv1beta1.StatusReady), c.EventuallyTimeout(time.Minute*10))
		Expect(snapshotExists()).To(BeTrue())

		c.Delete(rdsSnapshot)
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name(rdsSnapshot.Name), fetchSnapshot)
		}).ShouldNot(Succeed())

		Expect(snapshotExists()).To(BeFalse())
	})

	It("creates snapshot with a ttl", func() {
		c := helpers.TestClient
		rdsSnapshot.Spec.TTL = time.Minute * 10
		c.Create(rdsSnapshot)

		fetchSnapshot := &dbv1beta1.RDSSnapshot{}
		c.EventuallyGet(helpers.Name(rdsSnapshot.Name), fetchSnapshot, c.EventuallyStatus(dbv1beta1.StatusCreating))
		c.EventuallyGet(helpers.Name(rdsSnapshot.Name), fetchSnapshot, c.EventuallyStatus(dbv1beta1.StatusReady), c.EventuallyTimeout(time.Minute*10))
		Expect(snapshotExists()).To(BeTrue())

		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name(rdsSnapshot.Name), fetchSnapshot)
		}, time.Minute*5).ShouldNot(Succeed())

		Expect(snapshotExists()).To(BeFalse())
	})

})

func snapshotExists() bool {
	_, err := rdssvc.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: aws.String(rdsSnapshot.Spec.SnapshotID),
	})
	if err != nil {
		return false
	}
	return true
}

// Setup/teardown
func setupRDSInstance(rdsInstanceName string) (*string, error) {
	if rdsInstanceID != nil {
		return rdsInstanceID, nil
	}

	rawPassword := make([]byte, 32)
	rand.Read(rawPassword)
	password := make([]byte, base64.RawURLEncoding.EncodedLen(32))
	base64.RawURLEncoding.Encode(password, rawPassword)

	_, err := rdssvc.CreateDBInstance(&rds.CreateDBInstanceInput{
		StorageType:          aws.String("gp2"),
		AllocatedStorage:     aws.Int64(100),
		DBInstanceClass:      aws.String("db.t3.micro"),
		Engine:               aws.String("postgres"),
		DBInstanceIdentifier: aws.String(rdsInstanceName),
		MasterUsername:       aws.String("test_rds"),
		MasterUserPassword:   aws.String(string(password)),
	})
	if err != nil {
		return nil, err
	}

	for true {
		time.Sleep(time.Minute)
		describeDBInstancesOutput, err := rdssvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(rdsInstanceName),
		})
		if err != nil {
			return nil, err
		}
		dbStatus := aws.StringValue(describeDBInstancesOutput.DBInstances[0].DBInstanceStatus)
		if dbStatus == "available" {
			rdsInstanceID = describeDBInstancesOutput.DBInstances[0].DBInstanceIdentifier
			return nil, nil
		}
		if dbStatus == "error" {
			return nil, errors.New("rds instance in an error state")
		}
	}
	return &rdsInstanceName, nil
}
