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
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform NewRelic Component", func() {
	var comp components.Component

	Context("with newrelic template", func() {
		BeforeEach(func() {
			comp = summoncomponents.NewNewRelic()
			os.Setenv("NEW_RELIC_LICENSE_KEY", "1234asdf")
		})

		It("creates a config when enabled", func() {
			val := true
			instance.Spec.EnableNewRelic = &val
			Expect(comp).To(ReconcileContext(ctx))

			secret := &corev1.Secret{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev.newrelic", Namespace: "summon-dev"}, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data).To(HaveKeyWithValue("newrelic.ini", ContainSubstring("[newrelic]\nlicense_key = 1234asdf\napp_name = foo-dev-summon-platform\n")))
		})

		It("does not create a config when disabled", func() {
			val := false
			instance.Spec.EnableNewRelic = &val
			Expect(comp).To(ReconcileContext(ctx))

			secret := &corev1.Secret{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev.newrelic", Namespace: "summon-dev"}, secret)
			Expect(err).To(HaveOccurred())
		})

		It("does not create a config when not specified", func() {
			instance.Spec.EnableNewRelic = nil
			Expect(comp).To(ReconcileContext(ctx))

			secret := &corev1.Secret{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev.newrelic", Namespace: "summon-dev"}, secret)
			Expect(err).To(HaveOccurred())
		})
	})
})
