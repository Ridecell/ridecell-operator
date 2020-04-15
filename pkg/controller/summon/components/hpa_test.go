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
	"k8s.io/apimachinery/pkg/types"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("HorizontalPodAutoscaler (hpa) Component", func() {
	var comp components.Component

	Context("when ReplicaSpecs.<component>Auto is true", func() {
		BeforeEach(func() {
			// since default doesn't run, pretend we had celerydAuto set.
			instance.Spec.Replicas.CelerydAuto = true
		})

		It("creates an celeryd-hpa", func() {
			comp = summoncomponents.NewHPA("celeryd/hpa.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))

			hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd-hpa", Namespace: instance.Namespace}, hpa)
			Expect(err).NotTo(HaveOccurred())
			Expect(hpa.Spec.ScaleTargetRef.Kind).To(Equal("Deployment"))
			Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal("foo-dev-celeryd"))
			Expect(hpa.Spec.Metrics[0].External.Metric.Name).To(Equal("ridecell:rabbitmq_summon_queue_messages_ready"))
		})
	})

	Context("when ReplicaSpecs.<component>Auto is false (default)", func() {
		It("does not create celeryd-hpa", func() {
			comp = summoncomponents.NewHPA("celeryd/hpa.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd-hpa", Namespace: instance.Namespace}, hpa)
			Expect(err).To(HaveOccurred())
		})
	})

	/* TODO: Test finalizer logic
	Context("when ReplicaSpecs.<component>Auto was true, but set to false", func() {
		It("cleans up hpa component", func() {

		})
	})
	*/
})
