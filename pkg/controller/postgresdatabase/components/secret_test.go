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
	"github.com/Ridecell/ridecell-operator/pkg/components"
	pdcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresdatabase/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("postgresdatabase Secret Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = pdcomponents.NewSecret()
	})

	It("does nothing and doesnt error", func() {
		instance.Spec.DbConfigRef.Namespace = "summon-dev"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"password": []byte("cross-namespace"),
			},
		}
		ctx.Client = fake.NewFakeClient(secret)
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("copies secret from target to current namespace", func() {
		instance.Spec.DbConfigRef.Namespace = "target"
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: "target"},
			Data: map[string][]byte{
				"password": []byte("cross-namespace"),
			},
		}
		ctx.Client = fake.NewFakeClient(secret)
		Expect(comp).To(ReconcileContext(ctx))
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "mysecret", Namespace: "summon-dev"}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Data).To(HaveKeyWithValue("password", []byte("cross-namespace")))
	})
})
