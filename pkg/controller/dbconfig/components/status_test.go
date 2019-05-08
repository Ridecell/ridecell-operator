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

var _ = Describe("DbConfig Status Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = dbccomponents.NewStatus()
	})

	It("it sets ready when the database is exclusive", func() {
		instance.Spec.Postgres.Mode = "Exclusive"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
	})

	It("it sets ready when the database is shared and ready", func() {
		instance.Spec.Postgres.Mode = "Shared"
		instance.Status.Postgres.Status = "Ready"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
	})

	It("it does not set ready when the database is shared and not ready", func() {
		instance.Spec.Postgres.Mode = "Shared"
		instance.Status.Postgres.Status = "Error"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(""))
	})
})
