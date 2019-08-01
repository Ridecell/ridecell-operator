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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform Configmap Component", func() {
	Context("with no config", func() {
		It("creates a blank config", func() {
			comp := summoncomponents.NewConfigMap("configmap.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))

			configmap := &corev1.ConfigMap{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-config", Namespace: "default"}, configmap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configmap.Data).To(HaveKey("summon-platform.yml"))
			Expect(configmap.Data["summon-platform.yml"]).To(Equal("{}\n"))
		})
	})

	Context("with no config values", func() {
		It("creates a blank config", func() {
			instance.Spec.Config = map[string]summonv1beta1.ConfigValue{}

			comp := summoncomponents.NewConfigMap("configmap.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))

			configmap := &corev1.ConfigMap{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-config", Namespace: "default"}, configmap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configmap.Data).To(HaveKey("summon-platform.yml"))
			Expect(configmap.Data["summon-platform.yml"]).To(Equal("{}\n"))
		})
	})

	Context("with a string config value", func() {
		It("creates a config file", func() {
			instance.Spec.Config = map[string]summonv1beta1.ConfigValue{}
			val := "bar"
			instance.Spec.Config["foo"] = summonv1beta1.ConfigValue{String: &val}

			comp := summoncomponents.NewConfigMap("configmap.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))

			configmap := &corev1.ConfigMap{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-config", Namespace: "default"}, configmap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configmap.Data).To(HaveKey("summon-platform.yml"))
			Expect(configmap.Data["summon-platform.yml"]).To(Equal("{\"foo\":\"bar\"}\n"))
		})
	})

	Context("with an int config value", func() {
		It("creates a config file", func() {
			instance.Spec.Config = map[string]summonv1beta1.ConfigValue{}
			val := int32(42)
			instance.Spec.Config["foo"] = summonv1beta1.ConfigValue{Int: &val}

			comp := summoncomponents.NewConfigMap("configmap.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))

			configmap := &corev1.ConfigMap{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-config", Namespace: "default"}, configmap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configmap.Data).To(HaveKey("summon-platform.yml"))
			Expect(configmap.Data["summon-platform.yml"]).To(Equal("{\"foo\":42}\n"))
		})
	})

	Context("with a bool config value", func() {
		It("creates a config file", func() {
			instance.Spec.Config = map[string]summonv1beta1.ConfigValue{}
			val := true
			instance.Spec.Config["foo"] = summonv1beta1.ConfigValue{Bool: &val}

			comp := summoncomponents.NewConfigMap("configmap.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))

			configmap := &corev1.ConfigMap{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-config", Namespace: "default"}, configmap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configmap.Data).To(HaveKey("summon-platform.yml"))
			Expect(configmap.Data["summon-platform.yml"]).To(Equal("{\"foo\":true}\n"))
		})
	})
})
