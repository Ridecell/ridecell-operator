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

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	migrationcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/migrationjob/components"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MigrationJob Component", func() {
	comp := migrationcomponents.NewMigrationJob()

	BeforeEach(func() {
		comp = migrationcomponents.NewMigrationJob()
	})

	Describe(".Reconcile()", func() {
		Context("with no job existing", func() {
			It("creates a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Spec.Template.Spec.Containers[0].Name).To(Equal("test-job"))
				Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("image-name"))
			})

			It("checks template for a command", func() {
				instance.Spec.Template.Spec.Containers[0].Command[0] = "yes"
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Spec.Template.Spec.Containers[0].Command[0]).To(Equal("yes"))
			})
		})

		Context("with a running migration job", func() {
			BeforeEach(func() {
				job := &batchv1.Job{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo-dev-migrations",
						Namespace: "summon-dev",
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/version": "1.2.3",
								},
							},
						},
					},
				}
				err := ctx.Client.Create(context.TODO(), job)
				Expect(err).NotTo(HaveOccurred())
			})

			It("still has a migration job", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusMigrating))
			})
		})

		Context("with a successful migration job", func() {
			It("deletes the migration", func() {
				Expect(comp).To(ReconcileContext(ctx))
				job := &batchv1.Job{}
				err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: "summon-dev"}, job)
				Expect(err).NotTo(HaveOccurred())
				job.Status.Succeeded = 1
				err = ctx.Update(context.TODO(), job)
				Expect(err).NotTo(HaveOccurred())

				Expect(comp).To(ReconcileContext(ctx))
				Expect(comp).To(ReconcileContext(ctx))
				// Pending controller-runtime #213
				jobs := &metav1.List{}
				err = ctx.Client.List(context.TODO(), &client.ListOptions{Raw: &metav1.ListOptions{TypeMeta: metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"}}}, jobs)
				Expect(err).NotTo(HaveOccurred())
				Expect(jobs.Items).To(BeEmpty())
				Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
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
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/version": "1.2.3",
								},
							},
						},
					},
					Status: batchv1.JobStatus{
						Failed: 1,
					},
				}
				err := ctx.Client.Create(context.TODO(), job)
				Expect(err).NotTo(HaveOccurred())
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
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app.kubernetes.io/version": "1.2.2",
								},
							},
						},
					},
					Status: batchv1.JobStatus{
						Failed: 1,
					},
				}
				err := ctx.Client.Create(context.TODO(), job)
				Expect(err).NotTo(HaveOccurred())
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
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{},
							},
						},
					},
					Status: batchv1.JobStatus{
						Failed: 1,
					},
				}
				err := ctx.Client.Create(context.TODO(), job)
				Expect(err).NotTo(HaveOccurred())
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
	})
})
