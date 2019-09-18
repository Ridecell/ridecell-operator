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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/client-go/kubernetes/scheme"

	monitorv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var instance *monitorv1beta1.AlertManagerConfig
var ctx *components.ComponentContext

func TestTemplates(t *testing.T) {
	RegisterFailHandler(Fail)
	err := apis.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	RunSpecs(t, "AlertManagerConfig Components Suite @unit")
}

var _ = BeforeEach(func() {
	// Set up default-y values for tests to use if they want.
	instance = &monitorv1beta1.AlertManagerConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "alertmanager-auth", Namespace: "default"},
		Spec: monitorv1beta1.AlertManagerConfigSpec{
			AlertManagerName:      "alertmanager-infra",
			AlertManagerNamespace: "default",
			Route:                 "{\"match_re\":{\"servicename\":\".*dev-foo-service.*\"},\"routes\":[{\"receiver\":\"foo-pd\",\"match\":{\"severity\":\"critical\"},\"continue\":true},{\"receiver\":\"foo-slack\"}]}",
			Receivers: []string{
				"{\"name\":\"foo-slack\",\"slack_configs\":[{\"send_resolved\":true,\"channel\":\"#test-alert\",\"color\":\"{{ template \\\"slack.ridecell.color\\\" . }}\",\"title\":\"{{ template \\\"slack.ridecell.title\\\" . }}\",\"text\":\"{{ template \\\"slack.ridecell.text\\\" . }}\",\"icon_emoji\":\"{{ template \\\"slack.ridecell.icon_emoji\\\" . }}\",\"actions\":[{\"type\":\"button\",\"text\":\"Runbook :green_book:\",\"url\":\"{{ (index .Alerts 0).Annotations.runbook }}\"},{\"type\":\"button\",\"text\":\"Silence :no_bell:\",\"url\":\"https://dummy/#/silences\"},{\"type\":\"button\",\"text\":\"Dashboard :grafana:\",\"url\":\"{{ (index .Alerts 0).Annotations.dashboard }}\"},{\"type\":\"button\",\"text\":\"Query :mag:\",\"url\":\"{{ (index .Alerts 0).GeneratorURL }}\"}]},{\"send_resolved\":true,\"channel\":\"#test\",\"color\":\"{{ template \\\"slack.ridecell.color\\\" . }}\",\"title\":\"{{ template \\\"slack.ridecell.title\\\" . }}\",\"text\":\"{{ template \\\"slack.ridecell.text\\\" . }}\",\"icon_emoji\":\"{{ template \\\"slack.ridecell.icon_emoji\\\" . }}\",\"actions\":[{\"type\":\"button\",\"text\":\"Runbook :green_book:\",\"url\":\"{{ (index .Alerts 0).Annotations.runbook }}\"},{\"type\":\"button\",\"text\":\"Silence :no_bell:\",\"url\":\"https://dummy/#/silences\"},{\"type\":\"button\",\"text\":\"Dashboard :grafana:\",\"url\":\"{{ (index .Alerts 0).Annotations.dashboard }}\"},{\"type\":\"button\",\"text\":\"Query :mag:\",\"url\":\"{{ (index .Alerts 0).GeneratorURL }}\"}]}]}",
				"{\"name\":\"foo-pd\",\"pagerduty_configs\":[{\"send_resolved\":true,\"routing_key\":\"secret\",\"client\":\"dummy\",\"client_url\":\"https://dummy\",\"description\":\"{{ template \\\"pagerduty.ridecell.description\\\" .}}\",\"severity\":\"{{ if .CommonLabels.severity }}{{ .CommonLabels.severity | toLower }}{{ else }}critical{{ end }}\"}]}",
			},
		},
	}

	ctx = components.NewTestContext(instance, nil)
})
