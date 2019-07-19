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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
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
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-migrations", Namespace: "default"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})

			It("checks template info for a presigned url", func() {
				instance.Spec.Flavor = "test-flavor"
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-migrations", Namespace: "default"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Contains(job.Spec.Template.Spec.Containers[0].Command[2], "https://ridecell-flavors.s3.us-west-2.amazonaws.com/test-flavor.json.bz2")).To(BeTrue())
			})

			It("makes sure loadflavor command is not loaded into template", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-migrations", Namespace: "default"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("python manage.py migrate"))
			})
		})

		Context("with a running migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-migrations",
						Namespace: "default",
						Labels:    map[string]string{"app.kubernetes.io/version": "1.2.3"},
					},
				}
				ctx.Client = fake.NewFakeClient(job)
			})

			It("still has a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-migrations", Namespace: "default"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a successful migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-migrations",
						Namespace: "default",
						Labels:    map[string]string{"app.kubernetes.io/version": "1.2.3"},
					},
					Status: batchv1.JobStatus{
						Succeeded: 1,
					},
				}
				ctx.Client = fake.NewFakeClient(job)
			})

			It("deletes the migration", func() {
				Expect(comp).To(ReconcileContext(ctx))
				jobs := &batchv1.JobList{}
				err := ctx.Client.List(context.TODO(), jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(instance.Status.MigrateVersion).To(Equal("1.2.3"))
			})
		})

		Context("with a failed migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-migrations",
						Namespace: "default",
						Labels:    map[string]string{"app.kubernetes.io/version": "1.2.3"},
					},
					Status: batchv1.JobStatus{
						Failed: 1,
					},
				}
				ctx.Client = fake.NewFakeClient(job)
			})

			It("leaves the migration", func() {
				Expect(comp).NotTo(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-migrations", Namespace: "default"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a failed migration job from a previous version", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-migrations",
						Namespace: "default",
						Labels:    map[string]string{"app.kubernetes.io/version": "1.2.2"},
					},
					Status: batchv1.JobStatus{
						Failed: 1,
					},
				}
				ctx.Client = fake.NewFakeClient(job)
			})

			It("deletes the migration and reques", func() {
				resp, err := comp.Reconcile(ctx)
				Expect(err).NotTo(HaveOccurred())

				jobs := &batchv1.JobList{}
				err = ctx.Client.List(context.TODO(), jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(resp.Requeue).To(BeTrue())
				Expect(instance.Status.MigrateVersion).To(Equal(""))
			})
		})

		Context("with a failed migration that has no version label", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-migrations",
						Namespace: "default",
						Labels:    map[string]string{},
					},
					Status: batchv1.JobStatus{
						Failed: 1,
					},
				}
				ctx.Client = fake.NewFakeClient(job)
			})

			It("deletes the migration and reques", func() {
				resp, err := comp.Reconcile(ctx)
				Expect(err).NotTo(HaveOccurred())

				jobs := &batchv1.JobList{}
				err = ctx.Client.List(context.TODO(), jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(resp.Requeue).To(BeTrue())
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
	})
})
