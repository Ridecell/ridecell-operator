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
	"fmt"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SummonPlatform Postgres Component", func() {

	BeforeEach(func() {
	})

	Describe("IsReconcilable", func() {
		It("should always be true", func() {
			comp := summoncomponents.NewPostgres("postgres.yml.tpl", "postgres_operator/postgresoperator.yml.tpl")
			ok := comp.IsReconcilable(ctx)
			Expect(ok).To(BeTrue())
		})

	})

	Describe("Reconcile", func() {
		It("creates a postgres object by default", func() {
			comp := summoncomponents.NewPostgres("postgres.yml.tpl", "postgres_operator/postgresoperator.yml.tpl")
			instance.Spec.Database.ExclusiveDatabase = true
			Expect(comp).To(ReconcileContext(ctx))

			fetchPostgres := &postgresv1.Postgresql{}
			err := ctx.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s-database", instance.Name), Namespace: instance.Namespace}, fetchPostgres)
			Expect(err).ToNot(HaveOccurred())
		})

		It("creates a postgresoperator object if shareddatabase is true", func() {
			comp := summoncomponents.NewPostgres("postgres.yml.tpl", "postgres_operator/postgresoperator.yml.tpl")
			instance.Spec.Database.SharedDatabaseName = "foobar"
			instance.Spec.Database.ExclusiveDatabase = false

			fakePostgresql := &postgresv1.Postgresql{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foobar-database",
					Namespace: instance.Namespace,
				},
				Spec: postgresv1.PostgresSpec{
					TeamID: "foobar",
				},
			}

			ctx.Client = fake.NewFakeClient(fakePostgresql)

			Expect(comp).To(ReconcileContext(ctx))

			fetchOperator := &dbv1beta1.PostgresOperatorDatabase{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, fetchOperator)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
