/*
Copyright 2018-2019 Ridecell, Inc.

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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform Migrations Component", func() {
	comp := summoncomponents.NewMigrations("migrations.yml.tpl")

	BeforeEach(func() {
		comp = summoncomponents.NewMigrations("migrations.yml.tpl")
		instance.Status.BackupVersion = instance.Spec.Version
	})

	Describe(".IsReconcilable()", func() {
		Context("with a zero status", func() {
			It("returns false", func() {
				ok := comp.IsReconcilable(ctx)
				Expect(ok).To(BeFalse())
			})
		})

		Context("with Postgres not ready", func() {
			BeforeEach(func() {
				instance.Status.PostgresStatus = ""
			})

			It("returns false", func() {
				ok := comp.IsReconcilable(ctx)
				Expect(ok).To(BeFalse())
			})
		})

		Context("with Postgres ready", func() {
			BeforeEach(func() {
				instance.Status.PostgresStatus = dbv1beta1.StatusReady
			})

			It("returns false", func() {
				ok := comp.IsReconcilable(ctx)
				Expect(ok).To(BeFalse())
			})
		})

		Context("with Postgres and pull secret ready", func() {
			BeforeEach(func() {
				instance.Status.PostgresStatus = dbv1beta1.StatusReady
				instance.Status.PullSecretStatus = secretsv1beta1.StatusReady
			})

			It("returns true", func() {
				ok := comp.IsReconcilable(ctx)
				Expect(ok).To(BeTrue())
			})
		})

		Context("with migrations already applied", func() {
			BeforeEach(func() {
				instance.Status.PostgresStatus = dbv1beta1.StatusReady
				instance.Status.PullSecretStatus = secretsv1beta1.StatusReady
				instance.Status.MigrateVersion = "1.2.3"
			})

			It("returns true", func() {
				ok := comp.IsReconcilable(ctx)
				Expect(ok).To(BeTrue())
			})
		})
	})

	Describe(".Reconcile()", func() {
		Context("with no migration job existing", func() {
			BeforeEach(func() {
				instance.Spec.Version = "1.2.3"
				instance.Status.BackupVersion = "1.2.3"
			})
			It("creates a migration", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})

			It("checks template info for flavor", func() {
				instance.Spec.Flavor = "test-flavor"
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.Flavor).To(Equal("test-flavor"))
			})

			It("makes sure loadflavor command is not loaded into template", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.Flavor).To(Equal(""))
			})
		})

		Context("with a running migration job", func() {
			BeforeEach(func() {
				migration := &dbv1beta1.Migration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev",
						Namespace: "summon-dev",
					},
					Status: dbv1beta1.MigrationStatus{
						Status: dbv1beta1.StatusMigrating,
					},
				}
				ctx.Client = fake.NewFakeClient(migration)
			})

			It("still has a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusMigrating))
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a successful migration job", func() {
			BeforeEach(func() {
				migration := &dbv1beta1.Migration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev",
						Namespace: "summon-dev",
					},
					Spec: dbv1beta1.MigrationSpec{
						Version: "1.2.3",
					},
					Status: dbv1beta1.MigrationStatus{
						Status: dbv1beta1.StatusReady,
					},
				}
				ctx.Client = fake.NewFakeClient(migration)
			})

			It("deletes the migration", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).To(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal("1.2.3"))
			})
		})

		Context("with a failed migration job", func() {
			BeforeEach(func() {
				migration := &dbv1beta1.Migration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev",
						Namespace: "summon-dev",
					},
					Spec: dbv1beta1.MigrationSpec{
						Version: "1.2.3",
					},
					Status: dbv1beta1.MigrationStatus{
						Status:  dbv1beta1.StatusError,
						Message: "test-error",
					},
				}
				ctx.Client = fake.NewFakeClient(migration)
			})

			It("leaves the migration", func() {
				Expect(comp).NotTo(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a MigrateVersion", func() {
			BeforeEach(func() {
				instance.Status.MigrateVersion = "1234-abcd-master"
			})

			It("tries to run the CORE-1540 fixup", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.NoCore1540Fixup).To(Equal(false))
			})

			It("honors the NoCore1540Fixup flag", func() {
				instance.Spec.NoCore1540Fixup = true
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.Migration{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.NoCore1540Fixup).To(Equal(true))
			})
		})
	})
})
