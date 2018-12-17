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

	rmqvcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmq_vhost/components"
)

var _ = Describe("RabbitmqVhost Defaults Component", func() {
	It("does nothing on a filled out object", func() {
		comp := rmqvcomponents.NewDefaults()
		_, err := comp.Reconcile(ctx)
		Expect(err).NotTo(HaveOccurred())
	})
})
