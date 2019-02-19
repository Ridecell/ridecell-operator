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

	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("SummonPlatform service Component", func() {

	It("creates an service object using redis template", func() {
		comp := summoncomponents.NewService("redis/service.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.Service{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-redis", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an service object using static template", func() {
		comp := summoncomponents.NewService("static/service.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.Service{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-static", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an service object using redis template", func() {
		comp := summoncomponents.NewService("celerybeat/service.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.Service{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-celerybeat", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an service object using web template", func() {
		comp := summoncomponents.NewService("web/service.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.Service{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-web", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an service object using daphne template", func() {
		comp := summoncomponents.NewService("daphne/service.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.Service{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-daphne", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})
})
