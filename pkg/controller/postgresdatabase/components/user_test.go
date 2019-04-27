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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	pdcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresdatabase/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("PostgresDatabase User Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = pdcomponents.NewUser()
		instance.Spec.Owner = "foo"
		instance.Status.Connection = dbv1beta1.PostgresConnection{
			Host:     "mydb",
			Username: "myuser",
			PasswordSecretRef: helpers.SecretRef{
				Name: "mysecret",
			},
		}
	})

	Describe("IsReconcilable", func() {
		It("does create a user if the database is ready", func() {
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

		It("does not create a user if the database isn't ready", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("does not create a user if skipuser is on", func() {
			instance.Spec.SkipUser = true
			instance.Status.DatabaseStatus = dbv1beta1.StatusReady
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})
	})

	It("creates a user", func() {
		Expect(comp).To(ReconcileContext(ctx))
		user := &dbv1beta1.PostgresUser{}
		err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, user)
		Expect(err).ToNot(HaveOccurred())
		Expect(user.Spec.Username).To(Equal("foo"))
		Expect(user.Spec.Connection.Host).To(Equal("mydb"))
		Expect(user.Spec.Connection.Username).To(Equal("myuser"))
		Expect(user.Spec.Connection.PasswordSecretRef.Name).To(Equal("mysecret"))
	})
})
