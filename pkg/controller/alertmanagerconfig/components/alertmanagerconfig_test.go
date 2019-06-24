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

	"k8s.io/apimachinery/pkg/types"

	monitorv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	amccomponents "github.com/Ridecell/ridecell-operator/pkg/controller/alertmanagerconfig/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("AlertManagerConfig Component", func() {
	comp := amccomponents.NewAlertManagerConfig()
	var defaultConfig *corev1.Secret
	spec := monitorv1beta1.AlertManagerConfigSpec{
		Data:                  map[string]string{},
		AlertManagerName:      "alertmanager-infra",
		AlertManagerNamespace: "default",
	}

	BeforeEach(func() {
		defaultConfig = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "alertmanager-infra-default",
				Namespace: instance.Namespace,
			},
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
		_ = ctx.Create(context.TODO(), defaultConfig)

	})

	It("creates a alertmanager config secret rule", func() {
		instance.Spec = spec
		Expect(comp).To(ReconcileContext(ctx))
		fconfig := &corev1.Secret{}
		err := ctx.Get(context.Background(), types.NamespacedName{Name: "alertmanager-alertmanager-infra", Namespace: "default"}, fconfig)
		Expect(err).ToNot(HaveOccurred())
		Expect(fconfig.Data).To(HaveKey("alertmanager.yaml"))

	})
})
