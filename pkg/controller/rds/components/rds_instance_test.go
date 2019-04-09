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

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockRDSDBClient struct {
	rdsiface.RDSAPI

	dbInstanceExists bool
	createdDB        bool
	modifiedDB       bool
}

var _ = Describe("rds aws Component", func() {
	comp := rdscomponents.NewRDSInstance()
	var mockRDS *mockRDSDBClient

	BeforeEach(func() {
		comp = rdscomponents.NewRDSInstance()
		mockRDS = &mockRDSDBClient{}
		comp.InjectRDSAPI(mockRDS)
		instance.Spec.SubnetGroupName = "test"
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

	It("runs basic reconcile", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockRDS.createdDB).To(BeTrue())

		fetchCredentials := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "test.rds.credentials", Namespace: "default"}, fetchCredentials)
		Expect(err).ToNot(HaveOccurred())

		Expect(string(fetchCredentials.Data["username"])).To(Equal("test-user"))
		Expect(string(fetchCredentials.Data["password"])).To(HaveLen(43))
	})

	It("already has a database and password", func() {
		mockRDS.dbInstanceExists = true
		instance.Status.InstanceID = "alreadyexists"

		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "test.rds.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("password"),
			},
		}
		err := ctx.Client.Create(context.TODO(), newSecret)
		Expect(err).ToNot(HaveOccurred())

		Expect(comp).To(ReconcileContext(ctx))
	})

	It("has a database and no password", func() {
		mockRDS.dbInstanceExists = true
		instance.Status.InstanceID = "alreadyexists"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockRDS.modifiedDB)
	})
})

// Mock aws functions below

func (m *mockRDSDBClient) DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	if m.dbInstanceExists {
		dbInstances := []*rds.DBInstance{
			&rds.DBInstance{
				Endpoint: &rds.Endpoint{
					Address: aws.String("endpoint.test"),
					Port:    aws.Int64(8675309),
				},
				MasterUsername:   aws.String("test-user"),
				DBInstanceStatus: aws.String("test"),
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
			Port:    aws.Int64(8675309),
		},
		MasterUsername:   aws.String("test-user"),
		DBInstanceStatus: aws.String("test"),
	}
	m.createdDB = true
	return &rds.CreateDBInstanceOutput{DBInstance: dbInstance}, nil
}

func (m *mockRDSDBClient) ModifyDBInstance(input *rds.ModifyDBInstanceInput) (*rds.ModifyDBInstanceOutput, error) {
	m.modifiedDB = true
	return &rds.ModifyDBInstanceOutput{}, nil
}
