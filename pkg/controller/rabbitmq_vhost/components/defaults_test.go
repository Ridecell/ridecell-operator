/*
Copyright 2018 Ridecell, Inc.

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
	rmqvcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmq_vhost/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("RabbitmqVhost Defaults Component", func() {
	comp := rmqvcomponents.NewDefaults()

	BeforeEach(func() {
		comp = rmqvcomponents.NewDefaults()
	})

	It("fills in a default vhost name", func() {
		instance.Spec = dbv1beta1.RabbitmqVhostSpec{}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.VhostName).To(Equal("foo"))
	})

	It("does nothing on a filled out object", func() {
		instance.Spec = dbv1beta1.RabbitmqVhostSpec{VhostName: "other"}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.VhostName).To(Equal("other"))
	})
})
