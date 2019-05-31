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
	//"fmt"
	"os"
	"time"

	//"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/awserr"
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

var _ = Describe("rds controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var rdssvc *rds.RDS
	var randOwnerPrefix string

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()

		rdsSnapshot = &dbv1beta1.RDSSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "snapshot-controller-test",
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
