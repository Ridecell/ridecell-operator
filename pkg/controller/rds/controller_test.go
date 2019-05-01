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
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sess *session.Session
var rdssvc *rds.RDS
var ec2svc *ec2.EC2
var rdsInstance *dbv1beta1.RDSInstance
var randOwnerPrefix string
var rdsInstanceName string

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

		randOwnerPrefix = os.Getenv("RAND_OWNER_PREFIX")
		if randOwnerPrefix == "" {
			panic("$RAND_OWNER_PREFIX not set, failing test")
		}

		rdsInstanceName = fmt.Sprintf("%s-test-rds", randOwnerPrefix)

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
		ec2svc = ec2.New(sess)

		rdsInstance = &dbv1beta1.RDSInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rdsInstanceName,
				Namespace: helpers.Namespace,
			},
		}
		// multiaz false should save some time in testing while being operationally similar to normal usage
		rdsInstance.Spec.MultiAZ = aws.Bool(false)
		rdsInstance.Spec.MaintenanceWindow = "Mon:00:00-Mon:01:00"
	})

	AfterEach(func() {
		// Delete object and see if it cleans up on its own
		c := helpers.TestClient

		c.Delete(rdsInstance)

		// Database deletion may take a long time
		Eventually(func() bool { return dbInstanceExists() }, time.Minute*15, time.Second*30).Should(BeFalse())
		Eventually(func() bool { return dbParameterGroupExists() }, time.Minute*2, time.Second*10).Should(BeFalse())
		Eventually(func() bool { return securityGroupExists() }, time.Minute*2, time.Second*10).Should(BeFalse())

		helpers.TeardownTest()
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient
		c.Create(rdsInstance)

		fetchRDS := &dbv1beta1.RDSInstance{}
		c.EventuallyGet(helpers.Name(rdsInstanceName), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusCreating), c.EventuallyTimeout(time.Minute*3))

		// This process should max out at roughly 10 minutes
		c.EventuallyGet(helpers.Name(rdsInstanceName), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusReady), c.EventuallyTimeout(time.Minute*10))

		fetchSecret := &corev1.Secret{}
		c.Get(helpers.Name(fmt.Sprintf("%s.rds-user-password", rdsInstanceName)), fetchSecret)
		Expect(string(fetchSecret.Data["password"])).To(HaveLen(43))

		testContext := components.NewTestContext(fetchSecret, nil)

		db, err := postgres.Open(testContext, &fetchRDS.Status.Connection)
		Expect(err).ToNot(HaveOccurred())
		err = runTestQuery(db)
		Expect(err).ToNot(HaveOccurred())

		// Test password recreation via deletion
		c.Delete(fetchSecret)
		c.EventuallyGet(helpers.Name(rdsInstanceName), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusModifying), c.EventuallyTimeout(time.Minute*2))
		c.EventuallyGet(helpers.Name(rdsInstanceName), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusReady), c.EventuallyTimeout(time.Minute*5))

		c.Get(helpers.Name(fmt.Sprintf("%s.rds-user-password", rdsInstanceName)), fetchSecret)
		Expect(string(fetchSecret.Data["password"])).To(HaveLen(43))

		db2, err := postgres.Open(testContext, &fetchRDS.Status.Connection)
		Expect(err).ToNot(HaveOccurred())
		err = runTestQuery(db2)
		Expect(err).ToNot(HaveOccurred())
		db.Close()
		db2.Close()

		// Lets edit the parameter group!
		fetchRDS.Spec.Parameters = map[string]string{
			"log_min_duration_statement": "5000",
		}
		c.Update(fetchRDS)

		Eventually(func() bool {
			params, err := getDBParameters()
			if err != nil {
				return false
			}
			for k, v := range fetchRDS.Spec.Parameters {
				if params[k] != v {
					return false
				}
			}
			return true
		}, time.Minute*4, time.Second*20).Should(BeTrue())

		fetchRDS.Spec.Parameters = map[string]string{}
		c.Update(fetchRDS)

		c.EventuallyGet(helpers.Name(rdsInstanceName), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusModifying), c.EventuallyTimeout(time.Minute*3))
		c.EventuallyGet(helpers.Name(rdsInstanceName), fetchRDS, c.EventuallyStatus(dbv1beta1.StatusReady), c.EventuallyTimeout(time.Minute*11))

		params, err := getDBParameters()
		Expect(err).ToNot(HaveOccurred())
		Expect(params["log_min_duration_statement"]).To(Equal(""))
	})
})

func runTestQuery(db *sql.DB) error {
	output, err := db.Query(`SELECT groname FROM pg_group WHERE (SELECT usesysid FROM pg_user WHERE usename = current_user)=ANY(grolist);`)
	if err != nil {
		return errors.Wrap(err, "db.query")
	}
	defer output.Close()

	var column0 string
	for output.Next() {
		err := output.Scan(&column0)
		if err != nil {
			return errors.Wrap(err, "output.scan")
		}
		if column0 != "rds_superuser" {
			return errors.New("user was not in rds_superuser group")
		}
	}

	return nil
}

func dbInstanceExists() bool {
	_, err := rdssvc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(rdsInstance.Spec.InstanceID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == rds.ErrCodeDBInstanceNotFoundFault {
			return false
		}
	}
	return true
}

func dbParameterGroupExists() bool {
	_, err := rdssvc.DescribeDBParameterGroups(&rds.DescribeDBParameterGroupsInput{
		DBParameterGroupName: aws.String(rdsInstance.Name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == rds.ErrCodeDBParameterGroupNotFoundFault {
			return false
		}
	}
	return true
}

func securityGroupExists() bool {
	describeSecurityGroupsOutput, err := ec2svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("group-name"),
				Values: []*string{aws.String(rdsInstance.Name)},
			},
		},
	})
	if err != nil {
		return true
	}

	if len(describeSecurityGroupsOutput.SecurityGroups) > 0 {
		return true
	}
	return false
}

func getDBParameters() (map[string]string, error) {
	output := map[string]string{}
	var dbParams []*rds.Parameter
	err := rdssvc.DescribeDBParametersPages(&rds.DescribeDBParametersInput{
		DBParameterGroupName: aws.String(rdsInstance.Name),
	}, func(page *rds.DescribeDBParametersOutput, lastPage bool) bool {
		dbParams = append(dbParams, page.Parameters...)
		// if items returned < default MaxItems
		return !(len(page.Parameters) < 100)
	})
	if err != nil {
		return nil, errors.Wrap(err, "rds: failed to describe db parameters for controller_test")
	}
	for _, parameter := range dbParams {
		output[aws.StringValue(parameter.ParameterName)] = aws.StringValue(parameter.ParameterValue)
	}
	return output, nil
}
