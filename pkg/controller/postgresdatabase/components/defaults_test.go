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

	"github.com/Ridecell/ridecell-operator/pkg/components"
	pdcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresdatabase/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("PostgresDatabase Defaults Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = pdcomponents.NewDefaults()
	})

	It("sets a default database name", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.DatabaseName).To(Equal("foo_dev"))
	})

	It("sets a default owner", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Owner).To(Equal("foo_dev"))
	})
})