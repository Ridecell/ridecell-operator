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

package components_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
)

var _ = Describe("HorizontalPodAutoscaler (hpa) Component", func() {
	var comp components.Component

	Context("when ReplicaSpecs.<component>Auto is true", func() {
		BeforeEach(func() {
			// since default doesn't run, pretend we had celerydAuto set.
			boolVal := true
			instance.Spec.Replicas.CelerydAuto = &boolVal
		})

		It("creates an celeryd-hpa", func() {
			comp = summoncomponents.NewHPA("celeryd/hpa.yml.tpl", func(s *summonv1beta1.SummonPlatform) bool { return *s.Spec.Replicas.CelerydAuto })
			Expect(comp).To(ReconcileContext(ctx))

			hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd-hpa", Namespace: instance.Namespace}, hpa)
			Expect(err).NotTo(HaveOccurred())
			Expect(hpa.Spec.ScaleTargetRef.Kind).To(Equal("Deployment"))
			Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal("foo-dev-celeryd"))
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("ridecell:rabbitmq_summon_celery_queue_scaler"))
		})
	})

	Context("when ReplicaSpecs.<component>Auto is false (default)", func() {
		BeforeEach(func() {
			// since default doesn't run, pretend we had celerydAuto set.
			boolVal := false
			instance.Spec.Replicas.CelerydAuto = &boolVal
		})

		It("does not create celeryd-hpa", func() {
			comp = summoncomponents.NewHPA("celeryd/hpa.yml.tpl", func(s *summonv1beta1.SummonPlatform) bool { return *s.Spec.Replicas.CelerydAuto })
			Expect(comp).To(ReconcileContext(ctx))
			hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd-hpa", Namespace: instance.Namespace}, hpa)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when ReplicaSpecs.<component>Auto was true, but set to false", func() {
		//var hpa *autoscalingv2beta2.HorizontalPodAutoscaler
		BeforeEach(func() {
			// ReplicaSpecs.*Auto are true
			boolVal := true
			instance.Spec.Replicas.CelerydAuto = &boolVal
		})

		It("cleans up celeryd hpa component", func() {
			hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-celeryd-hpa", Namespace: instance.Namespace},
			}
			comp = summoncomponents.NewHPA("celeryd/hpa.yml.tpl", func(s *summonv1beta1.SummonPlatform) bool { return *s.Spec.Replicas.CelerydAuto })
			// Simulate HPA object already existing.
			ctx.Client = fake.NewFakeClient(instance, hpa)

			// Turn off autocaling and check that reconcile results in deleted HPA object.
			bVal := false
			instance.Spec.Replicas.CelerydAuto = &bVal
			Expect(comp).To(ReconcileContext(ctx))
			celerydHpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd-hpa", Namespace: instance.Namespace}, celerydHpa)
			Expect(err).To(HaveOccurred())
		})
	})
})
