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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	dbccomponents "github.com/Ridecell/ridecell-operator/pkg/controller/dbconfig/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("DbConfig Defaults Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = dbccomponents.NewDefaults()
	})

	It("does nothing with just an RDS config", func() {
		instance.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{}
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("does nothing with just a Local config", func() {
		instance.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("fails with neither postgres config", func() {
		Expect(comp).ToNot(ReconcileContext(ctx))
	})

	It("fails with both postgres configs", func() {
		instance.Spec.Postgres.RDS = &dbv1beta1.RDSInstanceSpec{}
		instance.Spec.Postgres.Local = &dbv1beta1.LocalPostgresSpec{}
		Expect(comp).ToNot(ReconcileContext(ctx))
	})
})
