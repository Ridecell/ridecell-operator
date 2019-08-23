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
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_pagerduty"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_sumologic"
	pomonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	alertmconfig "github.com/prometheus/alertmanager/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("monitor controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		os.Setenv("PG_ROUTING_KEY", "ALLISWELL")
		os.Setenv("SUMO_MOCK_URL", "http://localhost:8083")
		os.Setenv("PG_MOCK_URL", "http://localhost:8082")
		fake_pagerduty.Run()
		fake_sumologic.Run()
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&monitoringv1beta1.MonitorList{})
			helpers.DebugList(&pomonitoringv1.PrometheusRuleList{})
		}
		helpers.TeardownTest()
	})

	It("Creating monitor kind", func() {
		c := helpers.TestClient
		instance := &monitoringv1beta1.Monitor{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace},
			Spec: monitoringv1beta1.MonitorSpec{
				ServiceName: "dev-foo",
				MetricAlertRules: []monitoringv1beta1.MetricAlertRule{
					{
						Alert:       "HighErrorRate",
						Expr:        `job:request_latency_seconds:mean5m{job=", "} > 0.5`,
						Labels:      map[string]string{"severity": "page"},
						Annotations: map[string]string{"summary": "High request latency"},
					},
				},
				Notify: monitoringv1beta1.Notify{
					Slack: []string{
						"#test-alert",
						"#test",
					},
					PagerdutyTeam: "myteam",
				},
				LogAlertRules: []monitoringv1beta1.LogAlertRule{
					monitoringv1beta1.LogAlertRule{
						Name:        "Look for bad things realtime",
						Description: "looking for bad thing",
						Query:       `_sourceCategory=microservices/prod/us/job-management/* ("cancel_job_if_vehicle_got_reserved_or_moved" AND "TASK_FAILED") OR ("cancel_single_job_vehicle_reserved_task" AND "TASK_FAILED") OR ("cancel_single_job_location_mismatch_task" AND "TASK_FAILED")`,
						Condition:   "gt",
						Threshold:   4,
						Schedule:    "* * 0 0 0 0",
						Range:       "-15m",
						Severity:    "info",
						Runbook:     "https://ridecell.quip.com/ajDsAmRnWFQE/Monitoring",
					},
				},
			},
		}
		c.Create(instance)
		// Check Prom rules from here
		rule := &pomonitoringv1.PrometheusRule{}
		c.EventuallyGet(helpers.Name("foo"), rule)
		Expect(rule.Spec.Groups).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules[0].Alert).To(Equal("HighErrorRate"))

		// Check alert config from here
		alertConfig := &monitoringv1beta1.AlertManagerConfig{}
		c.EventuallyGet(helpers.Name("alertmanagerconfig-foo"), alertConfig)
		Expect(alertConfig.Spec.Data).To(HaveKey("receiver"))
		// Check receiver correct slack channel name
		receiver := &alertmconfig.Receiver{}
		err := yaml.Unmarshal([]byte(alertConfig.Spec.Data["receiver"]), receiver)
		Expect(err).ToNot(HaveOccurred())
		Expect(receiver.SlackConfigs[0].Channel).To(Equal("#test-alert"))
		Expect(receiver.PagerdutyConfigs[0].Severity).To(Equal(`{{ .CommonLabels.severity }}`))
		//Check Route have correct Receiver name
		Expect(alertConfig.Spec.Data).To(HaveKey("routes"))
		route := &alertmconfig.Route{}
		err = yaml.Unmarshal([]byte(alertConfig.Spec.Data["routes"]), route)
		Expect(err).ToNot(HaveOccurred())
		Expect(route.Receiver).To(Equal("foo"))
		// Check correct & default route condition present
		Expect(route.MatchRE["servicename"]).Should(ContainSubstring(instance.Spec.ServiceName))
	})

	It("Creating monitor kind without notification", func() {
		c := helpers.TestClient
		instance := &monitoringv1beta1.Monitor{
			ObjectMeta: metav1.ObjectMeta{Name: "bar", Namespace: helpers.Namespace},
			Spec: monitoringv1beta1.MonitorSpec{
				ServiceName: "dev-bar",
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

		// Check Prom rules from here
		rule := &pomonitoringv1.PrometheusRule{}
		c.EventuallyGet(helpers.Name("bar"), rule)
		Expect(rule.Spec.Groups).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules[0].Alert).To(Equal("HighErrorRate"))
	})
})
