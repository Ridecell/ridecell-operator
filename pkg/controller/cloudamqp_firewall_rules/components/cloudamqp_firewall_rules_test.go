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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	cfrcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/cloudamqp_firewall_rules/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_cloudamqp"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("CLOUDAMQP Firewall Defaults Component", func() {
	var comp components.Component
	os.Setenv("CLOUDAMQP_TEST", "true")
	os.Setenv("CLOUDAMQP_FIREWALL", "true")
	os.Setenv("CLOUDAMQP_TEST_URL", "http://localhost:9099/api/security/firewall")
	os.Setenv("CLOUDAMQP_API_KEY", "1234567890")
	fake_cloudamqp.Run()

	BeforeEach(func() {
		comp = cfrcomponents.NewCloudamqpFirewallRule()
	})

	It("puts firewall rules to cloudamqp", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(fake_cloudamqp.IPList).To(ContainElement("0.0.0.0/0"))
		Expect(fake_cloudamqp.IPList).To(ContainElement("1.2.3.4/32"))
	})

	It("puts default firewall rule if CLOUDAMQP_FIREWALL is false", func() {
		os.Setenv("CLOUDAMQP_FIREWALL", "false")
		Expect(comp).To(ReconcileContext(ctx))
		Expect(fake_cloudamqp.IPList).To(ContainElement("0.0.0.0/0"))
		Expect(fake_cloudamqp.IPList).ToNot(ContainElement("1.2.3.4/32"))
	})

})
