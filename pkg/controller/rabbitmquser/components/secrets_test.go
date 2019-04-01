/*
Copyright 2019-2020 Ridecell, Inc.

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
	rmqucomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmquser/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("RabbitmqUser Component", func() {
	BeforeEach(func() {
	})

	It("Reconciles successfully", func() {
		comp := rmqucomponents.NewSecrets()
		Expect(comp).To(ReconcileContext(ctx))
	})
	It("Creating a secret <instance_name>.rabbitmq-user-password ", func() {
		comp := rmqucomponents.NewSecrets()
		Expect(comp).To(ReconcileContext(ctx))
		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo.rabbitmq-user-password", Namespace: "default"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())
		secretValue := string(fetchSecret.Data["password"])
		Ω(secretValue).Should(HaveLen(20))
	})
})
