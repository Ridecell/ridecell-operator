/*
Copyright 2020 Ridecell, Inc.

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
	"os"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	gcpprojectcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/gcpproject/components"
)

var _ = Describe("gcpproject Defaults Component", func() {

	BeforeEach(func() {
		os.Setenv("FIREBASE_DATABASE_DEFAULT_RULES", "not empty")
	})

	It("does nothing on a filled out object", func() {
		comp := gcpprojectcomponents.NewDefaults()
		trueBool := true
		instance.Spec.EnableFirebase = &trueBool
		instance.Spec.EnableBilling = &trueBool
		instance.Spec.EnableRealtimeDatabase = &trueBool
		instance.Spec.RealtimeDatabaseRules = `{"test": "true"}`

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.EnableFirebase).To(PointTo(BeTrue()))
		Expect(instance.Spec.EnableBilling).To(PointTo(BeTrue()))
		Expect(instance.Spec.EnableRealtimeDatabase).To(PointTo(BeTrue()))
		Expect(instance.Spec.RealtimeDatabaseRules).To(Equal(`{"test": "true"}`))
	})

	It("sets defaults", func() {
		comp := gcpprojectcomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))

		Expect(instance.Spec.EnableFirebase).To(PointTo(BeFalse()))
		Expect(instance.Spec.EnableBilling).To(PointTo(BeFalse()))
		Expect(instance.Spec.EnableRealtimeDatabase).To(PointTo(BeFalse()))
		Expect(instance.Spec.RealtimeDatabaseRules).To(Equal("not empty"))
	})

})
