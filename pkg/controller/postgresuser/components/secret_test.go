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
	postgresusercomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresuser/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("PostgresUser Secret Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = postgresusercomponents.NewSecret()
	})

	It("creates a secret with a random password", func() {
		Expect(comp).To(ReconcileContext(ctx))
		secret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo.postgres-user-password", Namespace: "default"}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Data["password"]).To(HaveLen(43))
	})

	It("update a secret with a random password when the value is blank", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.postgres-user-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte{},
			},
		}
		ctx.Client = fake.NewFakeClient(instance, secret)
		Expect(comp).To(ReconcileContext(ctx))
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo.postgres-user-password", Namespace: "default"}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Data["password"]).To(HaveLen(43))
	})

	It("does not update an existing password", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.postgres-user-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("asdfqwer"),
			},
		}
		ctx.Client = fake.NewFakeClient(instance, secret)
		Expect(comp).To(ReconcileContext(ctx))
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo.postgres-user-password", Namespace: "default"}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Data).To(HaveKeyWithValue("password", []byte("asdfqwer")))
	})

	It("fills in the secret info in the status", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Connection.PasswordSecretRef.Name).To(Equal("foo.postgres-user-password"))
		Expect(instance.Status.Connection.PasswordSecretRef.Key).To(Equal("password"))
	})
})
