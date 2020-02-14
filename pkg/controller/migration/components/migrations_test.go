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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	//dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	migrationcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/migration/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("Migration Migrations Component", func() {
	comp := migrationcomponents.NewMigrations("migrations.yml.tpl")

	BeforeEach(func() {
		comp = migrationcomponents.NewMigrations("migrations.yml.tpl")
		os.Setenv("AWS_ACCESS_KEY_ID", "garbage")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "garbage")
	})

	Describe(".Reconcile()", func() {
		Context("with no migration job existing", func() {
			BeforeEach(func() {
				instance.Spec.NoCore1540Fixup = true
			})
			It("creates a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
			})

			It("checks template info for a presigned url", func() {
				instance.Spec.Flavor = "test-flavor"
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(strings.Contains(job.Spec.Template.Spec.Containers[0].Command[2], "https://ridecell-flavors.s3.us-west-2.amazonaws.com/test-flavor.json.bz2")).To(BeTrue())
			})

			It("makes sure loadflavor command is not loaded into template", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("python manage.py migrate -v3"))
			})
		})

		Context("with a running migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev-migrations",
						Namespace: "summon-dev",
						Labels:    map[string]string{"app.kubernetes.io/version": "1.2.3"},
					},
				}
				ctx.Client = fake.NewFakeClient(job)
			})

			It("still has a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with a successful migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev-migrations",
						Namespace: "summon-dev",
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
				// Pending controller-runtime #213
				jobs := &metav1.List{}
				err := ctx.Client.List(context.TODO(), &client.ListOptions{Raw: &metav1.ListOptions{TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}}}, jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
			})
		})

		Context("with a failed migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev-migrations",
						Namespace: "summon-dev",
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
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with a failed migration job from a previous version", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev-migrations",
						Namespace: "summon-dev",
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

				// Pending controller-runtime #213
				jobs := &metav1.List{}
				err = ctx.Client.List(context.TODO(), &client.ListOptions{Raw: &metav1.ListOptions{TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}}}, jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(resp.Requeue).To(BeTrue())
			})
		})

		Context("with a failed migration that has no version label", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev-migrations",
						Namespace: "summon-dev",
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

				// Pending controller-runtime #213
				jobs := &metav1.List{}
				err = ctx.Client.List(context.TODO(), &client.ListOptions{Raw: &metav1.ListOptions{TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}}}, jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(resp.Requeue).To(BeTrue())
			})
		})

		Context("with a bad template", func() {
			It("returns an error", func() {
				comp := migrationcomponents.NewMigrations("foo")
				_, err := comp.Reconcile(ctx)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("core1540fixup", func() {
			It("tries to run the CORE-1540 fixup", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("if [ -f common/management/commands/core_1540_pre_migrate.py ]; then python manage.py core_1540_pre_migrate; fi && python manage.py migrate -v3"))
			})

			It("honors the NoCore1540Fixup flag", func() {
				instance.Spec.NoCore1540Fixup = true
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Spec.Template.Spec.Containers[0].Command[2]).To(Equal("python manage.py migrate -v3"))
			})
		})
	})
})
