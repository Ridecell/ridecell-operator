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

package postgres_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	dbconfig_controller "github.com/Ridecell/ridecell-operator/pkg/controller/dbconfig"
	dbccomponents "github.com/Ridecell/ridecell-operator/pkg/controller/dbconfig/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("Postgres Shared Component", func() {
	var comp components.Component
	var dbconfig *dbv1beta1.DbConfig
	var pqdb *dbv1beta1.PostgresDatabase

	BeforeEach(func() {
		comp = dbccomponents.NewPostgres("Shared")
		dbconfig = instance
		pqdb = &dbv1beta1.PostgresDatabase{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev", Namespace: "summon-dev"},
			Spec: dbv1beta1.PostgresDatabaseSpec{
				DbConfig: "summon-dev",
			},
		}
	})

	Context("with top being DbConfig", func() {
		It("does not try to reconcile on an exclusive database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &postgresv1.PostgresSpec{}
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.PostgresqlList{}
			err := ctx.List(context.Background(), &client.ListOptions{}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Items).To(BeEmpty())
		})

		It("creates a local database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &postgresv1.PostgresSpec{}
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.Postgresql{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-database", Namespace: "summon-dev"}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Spec.TeamID).To(Equal("summon-dev"))
			Expect(dbconfig.Status.Postgres.Connection.Host).To(Equal("summon-dev-database"))
		})

		It("creates an RDS database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
				MaintenanceWindow: "Mon:00:00-Mon:01:00",
			}
			Expect(comp).To(ReconcileContext(ctx))

			rds := &dbv1beta1.RDSInstance{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev", Namespace: "summon-dev"}, rds)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with top being PostgresDatabase", func() {
		BeforeEach(func() {
			comp = dbccomponents.NewPostgres("Exclusive")
			ctx = components.NewTestContext(pqdb, dbconfig_controller.Templates)
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
		})

		It("does not try to reconcile on an shared database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &postgresv1.PostgresSpec{}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.PostgresqlList{}
			err := ctx.List(context.Background(), &client.ListOptions{}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Items).To(BeEmpty())
		})

		It("creates a local database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &postgresv1.PostgresSpec{}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.Postgresql{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-database", Namespace: "summon-dev"}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Spec.TeamID).To(Equal("foo-dev"))
			Expect(pqdb.Status.Connection.Host).To(Equal("foo-dev-database"))
		})

		It("creates an RDS database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
				MaintenanceWindow: "Mon:00:00-Mon:01:00",
			}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			rds := &dbv1beta1.RDSInstance{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, rds)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
