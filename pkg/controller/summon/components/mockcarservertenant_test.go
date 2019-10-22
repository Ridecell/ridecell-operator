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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform MockCarServerTenant Component", func() {
	var comp components.Component

	Context("MockCarServerTenant", func() {
		BeforeEach(func() {
			comp = summoncomponents.NewMockCarServerTenant()
		})

		It("creates a MockCarServerTenant when enabled", func() {
			instance.Spec.EnableMockCarServer = true
			Expect(comp).To(ReconcileContext(ctx))
			mockTenant := &summonv1beta1.MockCarServerTenant{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, mockTenant)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes a MockCarServerTenant when disabled", func() {
			instance.Spec.EnableMockCarServer = false
			Expect(comp).To(ReconcileContext(ctx))
			mockTenant := &summonv1beta1.MockCarServerTenant{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, mockTenant)
			Expect(err).To(HaveOccurred())
		})
	})
})
