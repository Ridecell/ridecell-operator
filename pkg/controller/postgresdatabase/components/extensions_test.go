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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	pdcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresdatabase/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("PostgresDatabase Extensions Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = pdcomponents.NewExtensions()
		instance.Spec.DatabaseName = "foo_dev"
	})

	It("does nothing with no extensions", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusCreating))
	})

	It("creates three extensions", func() {
		instance.Spec.Extensions = map[string]string{
			"postgis":          "",
			"postgis_topology": "",
			"pg_trgm":          "",
		}
		Expect(comp).To(ReconcileContext(ctx))

		ext := &dbv1beta1.PostgresExtension{}

		err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-postgis", Namespace: "summon-dev"}, ext)
		Expect(err).ToNot(HaveOccurred())
		Expect(ext.Spec.ExtensionName).To(Equal("postgis"))
		Expect(ext.Spec.Version).To(Equal(""))
		Expect(ext.Spec.Database.Database).To(Equal("foo_dev"))

		err = ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-postgis-topology", Namespace: "summon-dev"}, ext)
		Expect(err).ToNot(HaveOccurred())
		Expect(ext.Spec.ExtensionName).To(Equal("postgis_topology"))

		err = ctx.Get(context.Background(), types.NamespacedName{Name: "foo-dev-pg-trgm", Namespace: "summon-dev"}, ext)
		Expect(err).ToNot(HaveOccurred())
		Expect(ext.Spec.ExtensionName).To(Equal("pg_trgm"))
	})

	It("sets the status to creating", func() {
		ext := &dbv1beta1.PostgresExtension{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-postgis", Namespace: "summon-dev"},
			Status: dbv1beta1.PostgresExtensionStatus{
				Status: dbv1beta1.StatusReady,
			},
		}
		ctx.Client = fake.NewFakeClient(instance, ext)

		instance.Spec.Extensions = map[string]string{
			"postgis": "",
		}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusCreating))
	})
})
