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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform Superuser Component", func() {
	It("watches 1 type", func() {
		comp := summoncomponents.NewSuperuser()
		Expect(comp.WatchTypes()).To(HaveLen(1))
	})

	It("creates a superuser", func() {
		comp := summoncomponents.NewSuperuser()
		Expect(comp).To(ReconcileContext(ctx))

		user := &summonv1beta1.DjangoUser{}
		err := ctx.Client.Get(context.Background(), types.NamespacedName{Name: "foo-dev-dispatcher", Namespace: "summon-dev"}, user)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("with NoCreateSuperuser", func() {
		BeforeEach(func() { instance.Spec.NoCreateSuperuser = true })

		It("doesn't create the superuser", func() {
			comp := summoncomponents.NewSuperuser()
			Expect(comp).To(ReconcileContext(ctx))

			user := &summonv1beta1.DjangoUser{}
			err := ctx.Client.Get(context.Background(), types.NamespacedName{Name: "foo-dev-dispatcher", Namespace: "summon-dev"}, user)
			Expect(err).To(HaveOccurred())
			Expect(kerrors.IsNotFound(err)).To(BeTrue())
		})

		It("deletes an existing user", func() {
			user := &summonv1beta1.DjangoUser{ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-dispatcher", Namespace: "summon-dev"}}
			ctx.Client = fake.NewFakeClient(user)

			comp := summoncomponents.NewSuperuser()
			Expect(comp).To(ReconcileContext(ctx))

			err := ctx.Client.Get(context.Background(), types.NamespacedName{Name: "foo-dev-dispatcher", Namespace: "summon-dev"}, user)
			Expect(err).To(HaveOccurred())
			Expect(kerrors.IsNotFound(err)).To(BeTrue())
		})
	})
})
