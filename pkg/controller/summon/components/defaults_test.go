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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform Defaults Component", func() {
	It("does nothing on a filled out object", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{
			Hostname:              "foo.example.com",
			Environment:           "dev",
			WebReplicas:           intp(2),
			DaphneReplicas:        intp(2),
			ChannelWorkerReplicas: intp(2),
			StaticReplicas:        intp(2),
		}

		comp := summoncomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Hostname).To(Equal("foo.example.com"))
		Expect(instance.Spec.Environment).To(Equal("dev"))
		Expect(instance.Spec.WebReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.DaphneReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.ChannelWorkerReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.StaticReplicas).To(PointTo(BeEquivalentTo(2)))
	})

	It("sets a default hostname", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{
			WebReplicas:           intp(2),
			DaphneReplicas:        intp(2),
			ChannelWorkerReplicas: intp(2),
			StaticReplicas:        intp(2),
		}

		comp := summoncomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Hostname).To(Equal("foo.ridecell.us"))
	})

	It("sets a default pull secret", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{}

		comp := summoncomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.PullSecret).To(Equal("pull-secret"))
	})

	It("sets a default web replicas", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{
			DaphneReplicas:        intp(2),
			ChannelWorkerReplicas: intp(2),
			StaticReplicas:        intp(2),
		}

		comp := summoncomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.WebReplicas).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.DaphneReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.ChannelWorkerReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.StaticReplicas).To(PointTo(BeEquivalentTo(2)))
	})

	It("allows 0 web replicas", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{
			WebReplicas:           intp(0),
			DaphneReplicas:        intp(2),
			ChannelWorkerReplicas: intp(2),
			StaticReplicas:        intp(2),
		}

		comp := summoncomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.WebReplicas).To(PointTo(BeEquivalentTo(0)))
		Expect(instance.Spec.DaphneReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.ChannelWorkerReplicas).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.StaticReplicas).To(PointTo(BeEquivalentTo(2)))
	})

	It("Sets a default Secret for dev", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{}
		instance.Namespace = "dev"
		comp := summoncomponents.NewDefaults()

		_, err := comp.Reconcile(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.Spec.Secrets[0]).To(Equal("dev"))
		Expect(instance.Spec.Secrets[1]).To(Equal("foo"))
	})

	It("Sets a default Secret for prod", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{}
		instance.Namespace = "prod"
		comp := summoncomponents.NewDefaults()
		_, err := comp.Reconcile(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.Spec.Secrets[0]).To(Equal("prod"))
		Expect(instance.Spec.Secrets[1]).To(Equal("foo"))
	})

	It("Sets a default environment with summon prefix", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{}
		comp := summoncomponents.NewDefaults()
		instance.Namespace = "summon-dev"
		_, err := comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Spec.Environment).To(Equal("dev"))
	})

	It("Sets a default environment without the summon prefix", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{}
		comp := summoncomponents.NewDefaults()
		instance.Namespace = "dev"
		_, err := comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Spec.Environment).To(Equal("dev"))
	})
})
