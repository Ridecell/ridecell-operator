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

	"github.com/Ridecell/ridecell-operator/pkg/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SummonPlatform Postgres Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = summoncomponents.NewPostgres()
	})

	Describe("IsReconcilable", func() {
		It("should always be true", func() {
			ok := comp.IsReconcilable(ctx)
			Expect(ok).To(BeTrue())
		})

	})

	Describe("Reconcile", func() {
		It("creates a PostgresDatabase", func() {
			Expect(comp).To(ReconcileContext(ctx))

			db := &dbv1beta1.PostgresDatabase{}
			err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, db)
			Expect(err).ToNot(HaveOccurred())
		})

		It("sets PostgresStatus", func() {
			db := &dbv1beta1.PostgresDatabase{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-dev", Namespace: "summon-dev"},
				Status: dbv1beta1.PostgresDatabaseStatus{
					Status: dbv1beta1.StatusReady,
				},
			}
			ctx.Client = fake.NewFakeClient(db)

			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.PostgresStatus).To(Equal(dbv1beta1.StatusReady))
		})

		Context("with database name migration override", func() {
			BeforeEach(func() {
				instance.Spec.MigrationOverrides.PostgresDatabase = "legacy"
			})

			It("sets the DatabaseName correctly", func() {
				Expect(comp).To(ReconcileContext(ctx))
				db := &dbv1beta1.PostgresDatabase{}
				err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(db.Spec.DatabaseName).To(Equal("legacy"))
			})
		})

		Context("with database user migration override", func() {
			BeforeEach(func() {
				instance.Spec.MigrationOverrides.PostgresUsername = "legacy"
			})

			It("sets the DatabaseName correctly", func() {
				Expect(comp).To(ReconcileContext(ctx))
				db := &dbv1beta1.PostgresDatabase{}
				err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(db.Spec.Owner).To(Equal("legacy"))
			})
		})

		Context("with database ID migration override", func() {
			BeforeEach(func() {
				instance.Spec.MigrationOverrides.RDSInstanceID = "legacy"
			})

			It("sets the DatabaseName correctly", func() {
				Expect(comp).To(ReconcileContext(ctx))
				db := &dbv1beta1.PostgresDatabase{}
				err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(db.Spec.MigrationOverrides.RDSInstanceID).To(Equal("legacy"))
			})
		})

		Context("with database master username migration override", func() {
			BeforeEach(func() {
				instance.Spec.MigrationOverrides.RDSInstanceID = "legacy"
				instance.Spec.MigrationOverrides.RDSMasterUsername = "root"
			})

			It("sets the DatabaseName correctly", func() {
				Expect(comp).To(ReconcileContext(ctx))
				db := &dbv1beta1.PostgresDatabase{}
				err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(db.Spec.MigrationOverrides.RDSMasterUsername).To(Equal("root"))
			})
		})

		Context("with a DbConfigRef", func() {
			BeforeEach(func() {
				instance.Spec.Database.DbConfigRef.Name = "weirddb"
			})

			It("sets the DbConfigRef correctly", func() {
				Expect(comp).To(ReconcileContext(ctx))
				db := &dbv1beta1.PostgresDatabase{}
				err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, db)
				Expect(err).ToNot(HaveOccurred())
				Expect(db.Spec.DbConfigRef.Name).To(Equal("weirddb"))
			})
		})

	})
})
