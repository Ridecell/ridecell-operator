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
	k8sv1beta1 "k8s.io/api/extensions/v1beta1"
)

var _ = Describe("SummonPlatform ingress Component", func() {

	It("creates an ingress object using web template", func() {
		comp := summoncomponents.NewIngress("web/ingress.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &k8sv1beta1.Ingress{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-web", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an ingress object using static template", func() {
		comp := summoncomponents.NewIngress("static/ingress.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &k8sv1beta1.Ingress{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-static", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an ingress object using daphne template", func() {
		comp := summoncomponents.NewIngress("daphne/ingress.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &k8sv1beta1.Ingress{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-daphne", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates an ingress object with rules for given aliases using web template", func() {
		instance.Spec.Aliases = []string{"foo-1.ridecell.us", "foo-2.ridecell.us"}
		comp := summoncomponents.NewIngress("web/ingress.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &k8sv1beta1.Ingress{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-web", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
		Expect(target.Spec.Rules).To(HaveKeyWithValue("host", "foo.ridecell.us"))
		Expect(target.Spec.Rules).To(HaveKeyWithValue("host", "foo-1.ridecell.us"))
		Expect(target.Spec.Rules).To(HaveKeyWithValue("host", "foo-2.ridecell.us"))
	})
})
