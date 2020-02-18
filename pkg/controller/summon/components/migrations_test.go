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
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SummonPlatform Migrations Component", func() {
	comp := summoncomponents.NewMigrations("migrations.yml.tpl")

	BeforeEach(func() {
		comp = summoncomponents.NewMigrations("migrations.yml.tpl")
		os.Setenv("AWS_ACCESS_KEY_ID", "garbage")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "garbage")
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
			It("creates a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})

			It("checks template info for a presigned url", func() {
				instance.Spec.Flavor = "test-flavor"
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Contains(migration.Spec.Template.Spec.Containers[0].Command[2], "https://ridecell-flavors.s3.us-west-2.amazonaws.com/test-flavor.json.bz2")).To(BeTrue())
			})

			It("makes sure loadflavor command is not loaded into template", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("python manage.py migrate -v3"))
			})
		})

		Context("with an existing migration job", func() {
			BeforeEach(func() {
				migration := &dbv1beta1.MigrationJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev",
						Namespace: "summon-dev",
					},
				}
				ctx.Client = fake.NewFakeClient(migration)
			})

			It("has a not ready migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a successful migration job", func() {
			BeforeEach(func() {
				instance.Spec.Version = "1.2.3"
				migration := &dbv1beta1.MigrationJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev",
						Namespace: "summon-dev",
					},
					Status: dbv1beta1.MigrationJobStatus{
						Status: dbv1beta1.StatusReady,
					},
				}
				ctx.Client = fake.NewFakeClient(migration)
			})

			It("deletes the migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				// Pending controller-runtime #213
				jobs := &metav1.List{}
				err := ctx.Client.List(context.TODO(), &client.ListOptions{Raw: &metav1.ListOptions{TypeMeta: metav1.TypeMeta{APIVersion: "db.ridecell.io/v1beta1", Kind: "MigrationJob"}}}, jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(instance.Status.MigrateVersion).To(Equal("1.2.3"))
			})
		})

		Context("with a failed migration job", func() {
			BeforeEach(func() {
				migration := &dbv1beta1.MigrationJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev",
						Namespace: "summon-dev",
					},
					Status: dbv1beta1.MigrationJobStatus{
						Status: dbv1beta1.StatusError,
					},
				}
				ctx.Client = fake.NewFakeClient(migration)
			})

			It("leaves the migration", func() {
				Expect(comp).NotTo(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a bad template", func() {
			It("returns an error", func() {
				comp := summoncomponents.NewMigrations("foo")
				_, err := comp.Reconcile(ctx)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with a MigrateVersion", func() {
			BeforeEach(func() {
				instance.Status.MigrateVersion = "1234-abcd-master"
			})

			It("tries to run the CORE-1540 fixup", func() {
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("if [ -f common/management/commands/core_1540_pre_migrate.py ]; then python manage.py core_1540_pre_migrate; fi && python manage.py migrate -v3"))
			})

			It("honors the NoCore1540Fixup flag", func() {
				instance.Spec.NoCore1540Fixup = true
				Expect(comp).To(ReconcileContext(ctx))
				migration := &dbv1beta1.MigrationJob{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, migration)
				Expect(err).NotTo(HaveOccurred())
				Expect(migration.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("python manage.py migrate -v3"))
			})
		})
	})
})
