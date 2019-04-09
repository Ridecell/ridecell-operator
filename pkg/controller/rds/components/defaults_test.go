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

	rdscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rds/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("rds Defaults Component", func() {
	It("does nothing on a filled out object", func() {
		comp := rdscomponents.NewDefaults()
		instance.Spec.AllocatedStorage = 200
		instance.Spec.Engine = "test"
		instance.Spec.EngineVersion = "2"
		instance.Spec.InstanceClass = "db.m5.4xlarge"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.AllocatedStorage).To(Equal(int64(200)))
		Expect(instance.Spec.Engine).To(Equal("test"))
		Expect(instance.Spec.EngineVersion).To(Equal("2"))
		Expect(instance.Spec.InstanceClass).To(Equal("db.m5.4xlarge"))
	})

	It("sets defaults", func() {
		comp := rdscomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))

		Expect(instance.Spec.AllocatedStorage).To(Equal(int64(100)))
		Expect(instance.Spec.Engine).To(Equal("postgres"))
		Expect(instance.Spec.EngineVersion).To(Equal("11"))
		Expect(instance.Spec.InstanceClass).To(Equal("db.t3.micro"))
	})

})
