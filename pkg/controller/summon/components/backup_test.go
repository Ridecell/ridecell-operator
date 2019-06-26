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

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
	comp := summoncomponents.NewBackup("db/rdssnapshot.yml.tpl")
	postgresDatabase := &dbv1beta1.PostgresDatabase{}

	BeforeEach(func() {
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
		comp = summoncomponents.NewBackup("db/rdssnapshot.yml.tpl")
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
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
		fetchRDSSnapshot := &dbv1beta1.RDSSnapshot{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-1.2.3", Namespace: instance.Namespace}, fetchRDSSnapshot)
		Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	})

	It("tests snapshot creating status", func() {
		ctx.Client = fake.NewFakeClient(postgresDatabase)

	})
})
