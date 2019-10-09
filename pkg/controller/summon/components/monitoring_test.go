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

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/apimachinery/pkg/types"

	rmonitor "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
)

var _ = Describe("SummonPlatform monitoring Component", func() {
	var comp components.Component

	Context("Monitoring...", func() {
		BeforeEach(func() {
			comp = summoncomponents.NewMonitoring()

		})

		It("Is reconsining ", func() {
			instance.Spec.Monitoring.Enabled = true
			instance.Spec.Notifications.SlackChannel = "#test"
			instance.Spec.Notifications.Pagerdutyteam = "myteam"
			Expect(comp).To(ReconcileContext(ctx))

			monitor := &rmonitor.Monitor{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-monitoring", Namespace: "summon-dev"}, monitor)
			Expect(err).NotTo(HaveOccurred())
			Expect(monitor.Spec.Notify.Slack[0]).To(Equal("#test"))
			Expect(monitor.Spec.Notify.PagerdutyTeam).To(Equal("myteam"))
		})

		It("Missing slack should Reconcile without err", func() {
			instance.Spec.Monitoring.Enabled = true
			Expect(comp).To(ReconcileContext(ctx))
			// This will not create kind: monitor
			monitor := &rmonitor.Monitor{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-monitoring", Namespace: "summon-dev"}, monitor)
			Expect(err).To(HaveOccurred())
		})

	})
})
