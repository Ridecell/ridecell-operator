/*
Copyright 2020 Ridecell, Inc.

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

package migrationjob_test

import (
	"context"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const timeout = time.Second * 10

var _ = Describe("Migration controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var migration *dbv1beta1.MigrationJob

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		migration = &dbv1beta1.MigrationJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-dev",
				Namespace: helpers.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/version": "1.2.3",
				},
			},
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						RestartPolicy: "Never",
						Containers: []corev1.Container{
							corev1.Container{
								Name:    "test-job",
								Image:   "image-name",
								Command: []string{"yes"},
							},
						},
					},
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.kubernetes.io/version": "1.2.3",
						},
					},
				},
			},
		}
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&dbv1beta1.MigrationJobList{})
		}
		helpers.TeardownTest()
	})

	It("Creates a migration job", func() {
		err := helpers.Client.Create(context.TODO(), migration)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, timeout).Should(Succeed())

		// Makes sure status is set to migrating
		Eventually(func() bool {
			fetchMigration := &dbv1beta1.MigrationJob{}
			err = helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: helpers.Namespace}, fetchMigration)
			if err != nil {
				return false
			}
			return fetchMigration.Status.Status == dbv1beta1.StatusMigrating
		}, timeout).Should(Succeed())
	})

	It("finalizers delete job before deleting self", func() {
		err := helpers.Client.Create(context.TODO(), migration)
		Expect(err).NotTo(HaveOccurred())

		// Make sure job is created
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, timeout).Should(Succeed())

		err = helpers.Client.Delete(context.TODO(), migration)
		Expect(err).NotTo(HaveOccurred())

		// Deleted job
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, timeout).ShouldNot(Succeed())

		// Deleted self
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: helpers.Namespace}, &dbv1beta1.MigrationJob{})
		}, timeout).ShouldNot(Succeed())
	})

	It("has a successful job", func() {
		err := helpers.Client.Create(context.TODO(), migration)
		Expect(err).NotTo(HaveOccurred())

		job := &batchv1.Job{}
		// Make sure job is created
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, timeout).Should(Succeed())

		// Mark job as succeeded
		job.Status.Succeeded = 1
		err = helpers.Client.Update(context.TODO(), job)
		Expect(err).NotTo(HaveOccurred())

		// Checks that job is deleted
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, timeout).ShouldNot(Succeed())

		// Check that status is updated as expected
		fetchMigration := &dbv1beta1.MigrationJob{}
		err = helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: helpers.Namespace}, fetchMigration)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetchMigration.Status.Status).To(Equal(dbv1beta1.StatusReady))
	})

	It("has a failed job", func() {
		err := helpers.Client.Create(context.TODO(), migration)
		Expect(err).NotTo(HaveOccurred())

		job := &batchv1.Job{}
		// Make sure job is created
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, timeout).Should(Succeed())

		// Mark job as failed
		job.Status.Failed = 1
		err = helpers.Client.Update(context.TODO(), job)
		Expect(err).NotTo(HaveOccurred())

		// Checks that job is NOT deleted
		Consistently(func() error {
			return helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-migrations", Namespace: helpers.Namespace}, &batchv1.Job{})
		}, time.Second*5).Should(Succeed())

		// Check that status is updated as expected
		fetchMigration := &dbv1beta1.MigrationJob{}
		err = helpers.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: helpers.Namespace}, fetchMigration)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetchMigration.Status.Status).To(Equal(dbv1beta1.StatusError))
	})
})
