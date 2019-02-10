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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	djangousercomponents "github.com/Ridecell/ridecell-operator/pkg/controller/djangouser/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("DjangoUser Secret Component", func() {
	It("creates a password if no secret exists", func() {
		comp := djangousercomponents.NewSecret()
		Expect(comp).To(ReconcileContext(ctx))
		secret := &corev1.Secret{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo.example.com.django-password", Namespace: "default"}, secret)
		Expect(err).NotTo(HaveOccurred())
		password, ok := secret.Data["password"]
		Expect(ok).To(BeTrue())
		Expect(len(password)).To(Equal(22))
	})

	It("creates a password if the secret exists but has no password key", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com.django-password", Namespace: "default"},
		}
		ctx.Client = fake.NewFakeClient(secret)

		comp := djangousercomponents.NewSecret()
		Expect(comp).To(ReconcileContext(ctx))
		newSecret := &corev1.Secret{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo.example.com.django-password", Namespace: "default"}, newSecret)
		Expect(err).NotTo(HaveOccurred())
		password, ok := newSecret.Data["password"]
		Expect(ok).To(BeTrue())
		Expect(len(password)).To(Equal(22))
	})

	It("does not change an existing password", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.example.com.django-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("foo"),
			},
		}
		ctx.Client = fake.NewFakeClient(secret)

		comp := djangousercomponents.NewSecret()
		Expect(comp).To(ReconcileContext(ctx))
		newSecret := &corev1.Secret{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo.example.com.django-password", Namespace: "default"}, newSecret)
		Expect(err).NotTo(HaveOccurred())
		password, ok := newSecret.Data["password"]
		Expect(ok).To(BeTrue())
		Expect(password).To(Equal([]byte("foo")))
	})
})
