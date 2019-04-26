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

package components_test

import (
	"context"
	"database/sql"
	"fmt"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/dbpool"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockRDSDBClient struct {
	rdsiface.RDSAPI

	dbInstanceExists  bool
	hasTags           bool
	createdDB         bool
	modifiedDB        bool
	deletedDBInstance bool
	addedTags         bool
	dbStatus          string
}

var passwordSecret *corev1.Secret

var _ = Describe("rds aws Component", func() {
	comp := rdscomponents.NewRDSInstance()
	var mockRDS *mockRDSDBClient
	var dbMock sqlmock.Sqlmock
	var db *sql.DB

	BeforeEach(func() {
		var err error
		comp = rdscomponents.NewRDSInstance()
		mockRDS = &mockRDSDBClient{}
		comp.InjectRDSAPI(mockRDS)
		instance.Spec.SubnetGroupName = "test"
		instance.Status.Connection = dbv1beta1.PostgresConnection{
			Host:     "test-database",
			Port:     int(5432),
			Username: "test",
			Database: "test",
			PasswordSecretRef: helpers.SecretRef{
				Name: "test.rds-user-password",
				Key:  "password",
			},
		}
		passwordSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test.rds-user-password",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"password": []byte("test"),
			},
		}
		ctx.Client.Create(context.TODO(), passwordSecret)

		db, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		dbpool.Dbs.Store("postgres host=test-database port=5432 dbname=test user=test password='test' sslmode=require", db)
	})

	AfterEach(func() {
		db.Close()
		dbpool.Dbs.Delete("postgres host=test-database port=5432 dbname=test user=test password='test' sslmode=require")

		// Check for any unmet expectations.
		err := dbMock.ExpectationsWereMet()
		if err != nil {
			Fail(fmt.Sprintf("there were unfulfilled database expectations: %s", err))
		}
	})

	Describe("isReconcilable", func() {
		It("returns false", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("fails sg status check", func() {
			instance.Status.ParameterGroupStatus = dbv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("returns true", func() {
			instance.Status.SecurityGroupStatus = dbv1beta1.StatusReady
			instance.Status.ParameterGroupStatus = dbv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("creates a database", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.database.finalizer"))
		Expect(mockRDS.createdDB).To(BeTrue())
		Expect(mockRDS.modifiedDB).To(BeFalse())
		Expect(mockRDS.deletedDBInstance).To(BeFalse())
		Expect(mockRDS.addedTags).To(BeFalse())
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusCreating))
	})

	It("has a database in available state", func() {
		instance.Status.Status = dbv1beta1.StatusReady
		mockRDS.dbInstanceExists = true
		mockRDS.hasTags = true
		mockRDS.dbStatus = "available"

		dbMock.ExpectQuery("SELECT 1;").WillReturnRows()

		Expect(comp).To(ReconcileContext(ctx))

		Expect(instance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.database.finalizer"))
		Expect(mockRDS.modifiedDB).To(BeFalse())
		Expect(mockRDS.createdDB).To(BeFalse())
		Expect(mockRDS.deletedDBInstance).To(BeFalse())
		Expect(mockRDS.addedTags).To(BeFalse())
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
	})

	It("has a database in pending-reboot state", func() {
		instance.Status.Status = dbv1beta1.StatusReady
		mockRDS.dbInstanceExists = true
		mockRDS.hasTags = true
		mockRDS.dbStatus = "pending-reboot"

		dbMock.ExpectQuery("SELECT 1;").WillReturnRows()

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.database.finalizer"))
		Expect(mockRDS.modifiedDB).To(BeFalse())
		Expect(mockRDS.createdDB).To(BeFalse())
		Expect(mockRDS.addedTags).To(BeFalse())
		Expect(mockRDS.deletedDBInstance).To(BeFalse())
	})

	It("has an incorrect password", func() {
		instance.Status.Status = dbv1beta1.StatusReady
		mockRDS.dbInstanceExists = true
		mockRDS.hasTags = true
		mockRDS.dbStatus = "available"

		dbMock.ExpectQuery("SELECT 1;").WillReturnError(&pq.Error{Code: "28P01"})

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.database.finalizer"))
		Expect(mockRDS.modifiedDB).To(BeTrue())
		Expect(mockRDS.createdDB).To(BeFalse())
		Expect(mockRDS.addedTags).To(BeFalse())
		Expect(mockRDS.deletedDBInstance).To(BeFalse())
	})

	It("has a database in creating state", func() {
		mockRDS.dbInstanceExists = true
		mockRDS.hasTags = true
		mockRDS.dbStatus = "creating"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.database.finalizer"))
		Expect(mockRDS.modifiedDB).To(BeFalse())
		Expect(mockRDS.createdDB).To(BeFalse())
		Expect(mockRDS.addedTags).To(BeFalse())
		Expect(mockRDS.deletedDBInstance).To(BeFalse())
	})

	It("has a database with no tags", func() {
		mockRDS.dbInstanceExists = true
		mockRDS.dbStatus = "available"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.ObjectMeta.Finalizers[0]).To(Equal("rdsinstance.database.finalizer"))
		Expect(mockRDS.modifiedDB).To(BeFalse())
		Expect(mockRDS.createdDB).To(BeFalse())
		Expect(mockRDS.addedTags).To(BeTrue())
		Expect(mockRDS.deletedDBInstance).To(BeFalse())
	})

	It("test finalizer behavior during deletion", func() {
		instance.ObjectMeta.Finalizers = []string{"rdsinstance.database.finalizer"}
		mockRDS.dbInstanceExists = true
		currentTime := metav1.Now()
		instance.ObjectMeta.SetDeletionTimestamp(&currentTime)

		Expect(comp).To(ReconcileContext(ctx))

		fetchRDSInstance := &dbv1beta1.RDSInstance{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "test", Namespace: "default"}, fetchRDSInstance)
		Expect(err).ToNot(HaveOccurred())
		Expect(mockRDS.modifiedDB).To(BeFalse())
		Expect(mockRDS.createdDB).To(BeFalse())
		Expect(mockRDS.deletedDBInstance).To(BeTrue())
		Expect(fetchRDSInstance.ObjectMeta.Finalizers).To(HaveLen(0))
	})
})

// Mock aws functions below

func (m *mockRDSDBClient) DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	if m.dbInstanceExists {
		dbInstances := []*rds.DBInstance{
			&rds.DBInstance{
				Endpoint: &rds.Endpoint{
					Address: aws.String("endpoint.test"),
					Port:    aws.Int64(5432),
				},
				MasterUsername:   aws.String("test-user"),
				DBInstanceStatus: aws.String(m.dbStatus),
			},
		}
		return &rds.DescribeDBInstancesOutput{DBInstances: dbInstances}, nil
	}
	return nil, awserr.New(rds.ErrCodeDBInstanceNotFoundFault, "", nil)
}

func (m *mockRDSDBClient) CreateDBInstance(input *rds.CreateDBInstanceInput) (*rds.CreateDBInstanceOutput, error) {
	dbInstance := &rds.DBInstance{
		Endpoint: &rds.Endpoint{
			Address: aws.String("endpoint.test"),
			Port:    aws.Int64(5432),
		},
		MasterUsername:   aws.String("test-user"),
		DBInstanceStatus: aws.String("creating"),
	}
	m.createdDB = true
	m.hasTags = true
	return &rds.CreateDBInstanceOutput{DBInstance: dbInstance}, nil
}

func (m *mockRDSDBClient) ModifyDBInstance(input *rds.ModifyDBInstanceInput) (*rds.ModifyDBInstanceOutput, error) {
	if input.MasterUserPassword != nil && aws.StringValue(input.MasterUserPassword) != string(passwordSecret.Data["password"]) {
		return nil, errors.New("mock_rds: received incorrect password in modify")
	}
	m.modifiedDB = true
	return &rds.ModifyDBInstanceOutput{}, nil
}

func (m *mockRDSDBClient) DeleteDBInstance(input *rds.DeleteDBInstanceInput) (*rds.DeleteDBInstanceOutput, error) {
	if aws.StringValue(input.DBInstanceIdentifier) != instance.Name {
		return nil, errors.New("mock_rds: instance identifier did not match expected value")
	}
	m.deletedDBInstance = true
	return &rds.DeleteDBInstanceOutput{}, nil
}

func (m *mockRDSDBClient) ListTagsForResource(input *rds.ListTagsForResourceInput) (*rds.ListTagsForResourceOutput, error) {
	if m.hasTags {
		tags := []*rds.Tag{
			&rds.Tag{
				Key:   aws.String("Ridecell-Operator"),
				Value: aws.String("true"),
			},
			&rds.Tag{
				Key:   aws.String("tenant"),
				Value: aws.String("test"),
			},
		}
		return &rds.ListTagsForResourceOutput{TagList: tags}, nil
	}
	return &rds.ListTagsForResourceOutput{}, nil
}

func (m *mockRDSDBClient) AddTagsToResource(input *rds.AddTagsToResourceInput) (*rds.AddTagsToResourceOutput, error) {
	m.addedTags = true
	return &rds.AddTagsToResourceOutput{}, nil
}
