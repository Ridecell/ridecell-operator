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

package monitor_test

import (
	pomonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("monitor controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&monitoringv1beta1.MonitorList{})
			helpers.DebugList(&pomonitoringv1.PrometheusRuleList{})
		}

		helpers.TeardownTest()
	})

	It("does a thing", func() {
		c := helpers.TestClient
		instance := &monitoringv1beta1.Monitor{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace},
			Spec: monitoringv1beta1.MonitorSpec{
				MetricAlertRules: []monitoringv1beta1.MetricAlertRule{
					{
						Alert:       "HighErrorRate",
						Expr:        `job:request_latency_seconds:mean5m{job=", "} > 0.5`,
						Labels:      map[string]string{"severity": "page"},
						Annotations: map[string]string{"summary": "High request latency"},
					},
				},
			},
		}
		c.Create(instance)

		rule := &pomonitoringv1.PrometheusRule{}
		c.EventuallyGet(helpers.Name("foo"), rule)
		Expect(rule.Spec.Groups).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules[0].Alert).To(Equal("HighErrorRate"))
	})
})
