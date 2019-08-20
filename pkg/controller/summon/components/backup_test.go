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
	"time"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SummonPlatform backup Component", func() {
	var comp components.Component
	postgresDatabase := &dbv1beta1.PostgresDatabase{}

	BeforeEach(func() {
		trueBool := true
		instance.Spec.Backup = summonv1beta1.BackupSpec{
			TTL:            metav1.Duration{Duration: time.Minute * 5},
			WaitUntilReady: &trueBool,
		}
		postgresDatabase = &dbv1beta1.PostgresDatabase{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
			Spec: dbv1beta1.PostgresDatabaseSpec{
				DbConfigRef: corev1.ObjectReference{
					Name:      instance.Name,
					Namespace: instance.Namespace,
				},
			},
			Status: dbv1beta1.PostgresDatabaseStatus{
				RDSInstanceID: "test-rds-instance",
			},
		}
		comp = summoncomponents.NewBackup()
	})

	It("runs a basic reconcile", func() {
		ctx.Client = fake.NewFakeClient(postgresDatabase)
		Expect(comp).To(ReconcileContext(ctx))
		fetchRDSSnapshot := &dbv1beta1.RDSSnapshot{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-1.2.3", Namespace: instance.Namespace}, fetchRDSSnapshot)
		Expect(err).ToNot(HaveOccurred())
	})

	It("reconciles with matching versions", func() {
		ctx.Client = fake.NewFakeClient(postgresDatabase)
		instance.Status.BackupVersion = "1.2.3"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusMigrating))
		fetchRDSSnapshot := &dbv1beta1.RDSSnapshot{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-1.2.3", Namespace: instance.Namespace}, fetchRDSSnapshot)
		Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	})

	It("tests snapshot creating status", func() {
		// Set snapshot status to creating
		rdsSnapshot := &dbv1beta1.RDSSnapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-1.2.3",
				Namespace: instance.Namespace,
			},
			Status: dbv1beta1.RDSSnapshotStatus{
				Status: dbv1beta1.StatusCreating,
			},
		}
		ctx.Client = fake.NewFakeClient(postgresDatabase, rdsSnapshot)

		Expect(comp).To(ReconcileContext(ctx))

		fetchRDSSnapshot := &dbv1beta1.RDSSnapshot{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-1.2.3", Namespace: instance.Namespace}, fetchRDSSnapshot)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusCreatingBackup))
		Expect(instance.Status.BackupVersion).ToNot(Equal(instance.Spec.Version))

		// Set snapshot status to ready
		rdsSnapshot.Status.Status = dbv1beta1.StatusReady
		err = ctx.Client.Update(ctx.Context, rdsSnapshot)
		Expect(err).ToNot(HaveOccurred())

		Expect(comp).To(ReconcileContext(ctx))

		err = ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-1.2.3", Namespace: instance.Namespace}, fetchRDSSnapshot)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusMigrating))
		Expect(instance.Status.BackupVersion).To(Equal(instance.Spec.Version))

		Expect(fetchRDSSnapshot.Spec.TTL).To(Equal(instance.Spec.Backup.TTL))

		Expect(fetchRDSSnapshot.Spec.RDSInstanceID).To(Equal(postgresDatabase.Status.RDSInstanceID))
	})

	It("does not wait until snapshot is ready", func() {
		falseBool := false
		instance.Spec.Backup.WaitUntilReady = &falseBool
		ctx.Client = fake.NewFakeClient(postgresDatabase)
		Expect(comp).To(ReconcileContext(ctx))
		fetchRDSSnapshot := &dbv1beta1.RDSSnapshot{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-1.2.3", Namespace: instance.Namespace}, fetchRDSSnapshot)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusMigrating))

		Expect(fetchRDSSnapshot.Spec.TTL).To(Equal(instance.Spec.Backup.TTL))
		Expect(fetchRDSSnapshot.Spec.RDSInstanceID).To(Equal(postgresDatabase.Status.RDSInstanceID))
	})
})
