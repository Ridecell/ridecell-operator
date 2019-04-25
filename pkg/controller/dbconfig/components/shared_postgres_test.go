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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	dbccomponents "github.com/Ridecell/ridecell-operator/pkg/controller/dbconfig/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("DbConfig SharedPostgres Component @unit", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = dbccomponents.NewSharedPostgres()
	})

	It("does not try to reconcile on an exclusive database", func() {
		instance.Spec.Postgres.Mode = "Exclusive"
		Expect(comp.IsReconcilable(ctx)).To(BeFalse())
	})

	It("creates a local database", func() {
		instance.Spec.Postgres.Local = &postgresv1.PostgresSpec{}
		Expect(comp).To(ReconcileContext(ctx))

		postgres := &postgresv1.Postgresql{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "summon-dev-database", Namespace: "summon-dev"}, postgres)
		Expect(err).ToNot(HaveOccurred())
		Expect(postgres.Spec.TeamID).To(Equal("summon-dev"))
	})

	It("creates an RDS database", func() {
		instance.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{
			MaintenanceWindow: "Mon:00:00-Mon:01:00",
		}
		Expect(comp).To(ReconcileContext(ctx))

		rds := &dbv1beta1.RDSInstance{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "summon-dev", Namespace: "summon-dev"}, rds)
		Expect(err).ToNot(HaveOccurred())
	})
})
