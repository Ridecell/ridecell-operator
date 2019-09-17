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
	spcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/shared_components/postgres"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("Postgres Shared Component", func() {
	var comp components.Component
	var dbconfig *dbv1beta1.DbConfig
	var pqdb *dbv1beta1.PostgresDatabase

	BeforeEach(func() {
		comp = spcomponents.NewPostgres("Shared")
		dbconfig = instance
		pqdb = &dbv1beta1.PostgresDatabase{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev", Namespace: "summon-dev"},
		}
	})

	Context("with top being DbConfig", func() {
		It("does not try to reconcile on an exclusive database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.PostgresqlList{}
			err := ctx.List(context.Background(), &client.ListOptions{}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Items).To(BeEmpty())
		})

		It("does not reconcile on an exclusive database, nor create a periscope user", func() {
			// the PostgresDatabase controller should create the periscope user
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).To(HaveOccurred())
			Expect(pguser).To(Equal(&dbv1beta1.PostgresUser{}))
		})

		It("creates a local database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.Postgresql{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-database", Namespace: "summon-dev"}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Spec.TeamID).To(Equal("summon-dev"))
			Expect(dbconfig.Status.Postgres.Connection.Host).To(Equal("summon-dev-database"))
		})

		It("creates a periscope PostgresUser for a local database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(pguser).ToNot(Equal(&dbv1beta1.PostgresUser{}))
			Expect(err).ToNot(HaveOccurred())
			// periscope user connection should have inherited the dbconfig connection
			Expect(dbconfig.Status.Postgres.Connection.Host).To(Equal("summon-dev-database"))
			Expect(pguser.Spec.Connection).To(Equal(dbconfig.Status.Postgres.Connection))
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

		It("creates a periscope PostgresUser for an RDS database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
				MaintenanceWindow: "Mon:00:00-Mon:01:00",
			}
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(pguser).ToNot(Equal(&dbv1beta1.PostgresUser{}))
			Expect(err).ToNot(HaveOccurred())
		})

		It("creates a local database without periscope user if NoCreatePeriscopeUser is true", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			dbconfig.Spec.NoCreatePeriscopeUser = true
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).To(HaveOccurred())
			Expect(pguser).To(Equal(&dbv1beta1.PostgresUser{}))
		})

		It("creates a rds database without periscope user if NoCreatePeriscopeUser is true", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
				MaintenanceWindow: "Mon:00:00-Mon:01:00",
			}
			dbconfig.Spec.NoCreatePeriscopeUser = true
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).To(HaveOccurred())
			Expect(pguser).To(Equal(&dbv1beta1.PostgresUser{}))
		})

		Context("with an RDS ID override", func() {
			BeforeEach(func() {
				dbconfig.Spec.Postgres.Mode = "Shared"
				dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
					MaintenanceWindow: "Mon:00:00-Mon:01:00",
				}
				dbconfig.Spec.NoCreatePeriscopeUser = true
				dbconfig.Spec.MigrationOverrides.RDSInstanceID = "legacy"
			})

			It("creates the RDS database with the override", func() {
				Expect(comp).To(ReconcileContext(ctx))

				rds := &dbv1beta1.RDSInstance{}
				err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev", Namespace: "summon-dev"}, rds)
				Expect(err).ToNot(HaveOccurred())
				Expect(rds.Spec.InstanceID).To(Equal("legacy"))
			})
		})

		Context("with an RDS username override", func() {
			BeforeEach(func() {
				dbconfig.Spec.Postgres.Mode = "Shared"
				dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
					MaintenanceWindow: "Mon:00:00-Mon:01:00",
				}
				dbconfig.Spec.NoCreatePeriscopeUser = true
				dbconfig.Spec.MigrationOverrides.RDSMasterUsername = "root"
			})

			It("creates the RDS database with the override", func() {
				Expect(comp).To(ReconcileContext(ctx))

				rds := &dbv1beta1.RDSInstance{}
				err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev", Namespace: "summon-dev"}, rds)
				Expect(err).ToNot(HaveOccurred())
				Expect(rds.Spec.Username).To(Equal("root"))
			})
		})
	})

	Context("with top being PostgresDatabase", func() {
		BeforeEach(func() {
			comp = spcomponents.NewPostgres("Exclusive")
			ctx.Top = pqdb
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
		})

		It("does not try to reconcile on an shared database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.PostgresqlList{}
			err := ctx.List(context.Background(), &client.ListOptions{}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Items).To(BeEmpty())
		})

		It("does not try to create a periscope user for shared database", func() {
			dbconfig.Spec.Postgres.Mode = "Shared"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "summon-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).To(HaveOccurred())
			Expect(pguser).To(Equal(&dbv1beta1.PostgresUser{}))
		})

		It("creates a local database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			postgres := &postgresv1.Postgresql{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-database", Namespace: "summon-dev"}, postgres)
			Expect(err).ToNot(HaveOccurred())
			Expect(postgres.Spec.TeamID).To(Equal("foo-dev"))
			Expect(pqdb.Status.AdminConnection.Host).To(Equal("foo-dev-database"))
		})

		It("creates a periscope user for a local database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).ToNot(HaveOccurred())
			Expect(pguser).ToNot(Equal(&dbv1beta1.PostgresUser{}))

			// periscope user connection should have inherited the postgresdatabase connection
			Expect(pguser.Spec.Connection).To(Equal(pqdb.Status.AdminConnection))
		})

		It("does not create a periscope user for local database if NoPeriscopeUser flag is set", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
			dbconfig.Spec.NoCreatePeriscopeUser = true
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).To(HaveOccurred())
			Expect(pguser).To(Equal(&dbv1beta1.PostgresUser{}))
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
			Expect(rds.Spec.InstanceID).To(Equal(""))
		})

		It("creates a periscope user for a RDS database", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
				MaintenanceWindow: "Mon:00:00-Mon:01:00",
			}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).ToNot(HaveOccurred())
			Expect(pguser).ToNot(Equal(&dbv1beta1.PostgresUser{}))

			// periscope user connection should have inherited the postgresdatabase connection
			Expect(pguser.Spec.Connection).To(Equal(pqdb.Status.AdminConnection))
		})

		It("does not create a periscope user for a RDS database if NoPeriscopeUser flag is set", func() {
			dbconfig.Spec.Postgres.Mode = "Exclusive"
			dbconfig.Spec.NoCreatePeriscopeUser = true
			dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
				MaintenanceWindow: "Mon:00:00-Mon:01:00",
			}
			ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			Expect(comp).To(ReconcileContext(ctx))

			pguser := &dbv1beta1.PostgresUser{}
			err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-periscope", Namespace: "summon-dev"}, pguser)
			Expect(err).To(HaveOccurred())
			Expect(pguser).To(Equal(&dbv1beta1.PostgresUser{}))
		})

		Context("with an RDS ID override", func() {
			BeforeEach(func() {
				dbconfig.Spec.Postgres.Mode = "Exclusive"
				dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
					MaintenanceWindow: "Mon:00:00-Mon:01:00",
				}
				pqdb.Spec.MigrationOverrides.RDSInstanceID = "legacy"
				ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			})

			It("creates the RDS database with the override", func() {
				ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
				Expect(comp).To(ReconcileContext(ctx))

				rds := &dbv1beta1.RDSInstance{}
				err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, rds)
				Expect(err).ToNot(HaveOccurred())
				Expect(rds.Spec.InstanceID).To(Equal("legacy"))
			})
		})

		Context("with an RDS username override", func() {
			BeforeEach(func() {
				dbconfig.Spec.Postgres.Mode = "Exclusive"
				dbconfig.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
					MaintenanceWindow: "Mon:00:00-Mon:01:00",
				}
				pqdb.Spec.MigrationOverrides.RDSMasterUsername = "root"
				ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
			})

			It("creates the RDS database with the override", func() {
				ctx.Client = fake.NewFakeClient(dbconfig, pqdb)
				Expect(comp).To(ReconcileContext(ctx))

				rds := &dbv1beta1.RDSInstance{}
				err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, rds)
				Expect(err).ToNot(HaveOccurred())
				Expect(rds.Spec.Username).To(Equal("root"))
			})
		})
	})

})
