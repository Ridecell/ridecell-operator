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

package alertmanagerconfig_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	alertconfig "github.com/prometheus/alertmanager/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("alertmanagerconfig controller", func() {
	var helpers *test_helpers.PerTestHelpers
	os.Setenv("PG_ROUTING_KEY", "foopdkey")

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		c := helpers.TestClient
		// Make a default config.
		defaultConfig := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "alertmanager-infra-default", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"alertmanager.yaml": []byte(`
global:
  resolve_timeout: 5m
  slack_api_url: https://hooks.slack.com/services/test123/test123
route:
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 2h
  receiver: 'test-alert'
  group_by: [alertname, pod]
receivers:
- name: 'test-alert'
  slack_configs:
  - channel: '#test-alert'
    send_resolved: true
    icon_url: "https://avatars3.githubusercontent.com/u/3380462"
    title: |-
    text: >-
      {{ range .Alerts -}}
        *Alert:* {{ .Annotations.title }}{{ if .Labels.severity }} - {{ .Labels.severity }}{{ end }}
      *Description:* {{ .Annotations.description }}
      *Details:*
        {{ range .Labels.SortedPairs }} • *{{ .Name }}:* {{ .Value }}
        {{ end }}
      {{ end }}
        `),
			},
		}
		c.Create(defaultConfig)

	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&monitoringv1beta1.AlertManagerConfigList{})

		}
		helpers.TeardownTest()
	})

	It("does a thing", func() {
		c := helpers.TestClient
		instance := &monitoringv1beta1.AlertManagerConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "alertmanager-alertmanager-infra", Namespace: helpers.Namespace},
			Spec: monitoringv1beta1.AlertManagerConfigSpec{
				AlertManagerName:      "alertmanager-infra",
				AlertManagerNamespace: helpers.Namespace,
				Route:                 "{\"match_re\":{\"servicename\":\".*dev-foo-service.*\"},\"routes\":[{\"receiver\":\"foo-pd\",\"match\":{\"severity\":\"critical\"},\"continue\":true},{\"receiver\":\"foo-slack\"}]}",
				Receivers: []string{
					"{\"name\":\"foo-slack\",\"slack_configs\":[{\"send_resolved\":true,\"channel\":\"#test-alert\",\"color\":\"{{ template \\\"slack.ridecell.color\\\" . }}\",\"title\":\"{{ template \\\"slack.ridecell.title\\\" . }}\",\"text\":\"{{ template \\\"slack.ridecell.text\\\" . }}\",\"icon_emoji\":\"{{ template \\\"slack.ridecell.icon_emoji\\\" . }}\",\"actions\":[{\"type\":\"button\",\"text\":\"Runbook :green_book:\",\"url\":\"{{ (index .Alerts 0).Annotations.runbook }}\"},{\"type\":\"button\",\"text\":\"Silence :no_bell:\",\"url\":\"https://dummy/#/silences\"},{\"type\":\"button\",\"text\":\"Dashboard :grafana:\",\"url\":\"{{ (index .Alerts 0).Annotations.dashboard }}\"},{\"type\":\"button\",\"text\":\"Query :mag:\",\"url\":\"{{ (index .Alerts 0).GeneratorURL }}\"}]},{\"send_resolved\":true,\"channel\":\"#test\",\"color\":\"{{ template \\\"slack.ridecell.color\\\" . }}\",\"title\":\"{{ template \\\"slack.ridecell.title\\\" . }}\",\"text\":\"{{ template \\\"slack.ridecell.text\\\" . }}\",\"icon_emoji\":\"{{ template \\\"slack.ridecell.icon_emoji\\\" . }}\",\"actions\":[{\"type\":\"button\",\"text\":\"Runbook :green_book:\",\"url\":\"{{ (index .Alerts 0).Annotations.runbook }}\"},{\"type\":\"button\",\"text\":\"Silence :no_bell:\",\"url\":\"https://dummy/#/silences\"},{\"type\":\"button\",\"text\":\"Dashboard :grafana:\",\"url\":\"{{ (index .Alerts 0).Annotations.dashboard }}\"},{\"type\":\"button\",\"text\":\"Query :mag:\",\"url\":\"{{ (index .Alerts 0).GeneratorURL }}\"}]}]}",
					"{\"name\":\"foo-pd\",\"pagerduty_configs\":[{\"send_resolved\":true,\"routing_key\":\"secret\",\"client\":\"dummy\",\"client_url\":\"https://dummy\",\"description\":\"{{ template \\\"pagerduty.ridecell.description\\\" .}}\",\"severity\":\"{{ if .CommonLabels.severity }}{{ .CommonLabels.severity | toLower }}{{ else }}critical{{ end }}\"}]}",
				},
			},
		}
		c.Create(instance)
		fconfig := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("alertmanager-alertmanager-infra"), fconfig)
		Expect(fconfig.Data).To(HaveKey("alertmanager.yaml"))
		config, err := alertconfig.Load(string(fconfig.Data["alertmanager.yaml"]))
		Expect(err).ToNot(HaveOccurred())
		Expect(len(config.Receivers)).Should(BeNumerically(">=", 2))
		Expect(string(config.Receivers[2].PagerdutyConfigs[0].RoutingKey)).Should(Equal(os.Getenv("PG_ROUTING_KEY")))
		Expect(config.Global.SlackAPIURL.String()).Should(Equal("https://hooks.slack.com/services/test123/test123"))

	})
})
