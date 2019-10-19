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

	"github.com/Ridecell/ridecell-operator/pkg/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
)

var _ = Describe("servicemonitor Component", func() {
	var comp components.Component

	BeforeEach(func() {
		intp := func(i int32) *int32 { return &i }
		replicas := &instance.Spec.Replicas

		replicas.Web = intp(1)
		replicas.Static = intp(1)
		replicas.Daphne = intp(1)
		replicas.ChannelWorker = intp(1)
		replicas.Celeryd = intp(1)
	})

	It("creates a non zero web pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("web/podDisruptionBudget.yml.tpl")
		instance.Spec.Replicas.Web = intp(2)
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("10%"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a web pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("web/podDisruptionBudget.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("0"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a static pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("static/podDisruptionBudget.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-static", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("0"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a non zero static pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("static/podDisruptionBudget.yml.tpl")
		instance.Spec.Replicas.Static = intp(2)
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-static", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("10%"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a daphne pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("daphne/podDisruptionBudget.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-daphne", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("0"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a non zero daphne pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("daphne/podDisruptionBudget.yml.tpl")
		instance.Spec.Replicas.Daphne = intp(2)
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-daphne", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("10%"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a channelworker pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("channelworker/podDisruptionBudget.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-channelworker", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("0"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a non zero channelworker pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("channelworker/podDisruptionBudget.yml.tpl")
		instance.Spec.Replicas.ChannelWorker = intp(2)
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-channelworker", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("10%"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a celeryd pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("celeryd/podDisruptionBudget.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("0"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})

	It("creates a non zero celeryd pod disruption budget", func() {
		comp = summoncomponents.NewPodDisruptionBudget("celeryd/podDisruptionBudget.yml.tpl")
		instance.Spec.Replicas.Celeryd = intp(2)
		Expect(comp).To(ReconcileContext(ctx))

		disruptionBudget := &policyv1beta1.PodDisruptionBudget{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-celeryd", Namespace: instance.Namespace}, disruptionBudget)
		Expect(err).ToNot(HaveOccurred())
		Expect(disruptionBudget.Spec.MaxUnavailable.String()).To(Equal("10%"))
		Expect(disruptionBudget.Spec.MaxUnavailable.IntValue()).To(Equal(0))
	})
})
