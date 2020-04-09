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

	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("VerticalPodAutoscaler (vpa) Component", func() {
	var comp components.Component

	It("creates a VPA for businessPortal component", func() {
		comp = summoncomponents.NewVPA("businessPortal/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-businessportal", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for celerybeat component", func() {
		comp = summoncomponents.NewVPA("celerybeat/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celerybeat", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("StatefulSet"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for celeryd component", func() {
		comp = summoncomponents.NewVPA("celeryd/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for channelworker component", func() {
		comp = summoncomponents.NewVPA("channelworker/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-channelworker", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for daphne component", func() {
		comp = summoncomponents.NewVPA("daphne/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-daphne", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for dispatch component", func() {
		comp = summoncomponents.NewVPA("dispatch/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-dispatch", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for hwaux component", func() {
		comp = summoncomponents.NewVPA("hwAux/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-hwaux", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for redis component", func() {
		comp = summoncomponents.NewVPA("redis/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-redis", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for static component", func() {
		comp = summoncomponents.NewVPA("static/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-static", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for tripShare component", func() {
		comp = summoncomponents.NewVPA("tripShare/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-tripshare", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

	It("creates a VPA for a web component", func() {
		comp = summoncomponents.NewVPA("web/vpa.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		vpa := &autoscalingv1.VerticalPodAutoscaler{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, vpa)
		Expect(err).NotTo(HaveOccurred())
		Expect(vpa.Spec.TargetRef.Kind).To(Equal("Deployment"))
		// vpa UpdateMode is a pointer to type UpdateMode, so need to derefence and compare to type
		Expect(*vpa.Spec.UpdatePolicy.UpdateMode).To(Equal(autoscalingv1.UpdateModeOff))
	})

})
