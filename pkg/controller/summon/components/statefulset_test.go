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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("SummonPlatform statefulset Component", func() {
	comp := summoncomponents.NewStatefulSet("celerybeat/statefulset.yml.tpl", true)

	BeforeEach(func() {
		comp = summoncomponents.NewStatefulSet("celerybeat/statefulset.yml.tpl", true)
	})

	Context("IsReconcilable", func() {

		It("fails first check", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("doesnt wait for database", func() {
			comp = summoncomponents.NewStatefulSet("celerybeat/statefulset.yml.tpl", false)
			instance.Status.PullSecretStatus = secretsv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

		It("fails first database check", func() {
			instance.Status.PullSecretStatus = secretsv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("passes all checks", func() {
			instance.Status.PullSecretStatus = secretsv1beta1.StatusReady
			instance.Status.PostgresStatus = dbv1beta1.StatusReady
			instance.Status.PostgresExtensionStatus = summonv1beta1.StatusReady
			instance.Status.MigrateVersion = instance.Spec.Version
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

	})

	It("creates an statefulset object using celerybeat template", func() {
		Expect(comp).To(ReconcileContext(ctx))
		target := &appsv1.StatefulSet{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-celerybeat", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})
})
