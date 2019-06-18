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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	alertconfig "github.com/prometheus/alertmanager/config"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("alertmanagerconfig controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		c := helpers.TestClient
		// Make a default config.
		defaultConfig := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "alertmanagerconfig-default", Namespace: helpers.Namespace},
			Data: map[string]string{
				"alertmanager.yml": "asdfasdfasdf",
			},
		}
		c.Create(defaultConfig)

	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("does a thing", func() {
		c := helpers.TestClient
		instance := &monitoringv1beta1.AlertManagerConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace},
			Spec:       monitoringv1beta1.AlertManagerConfigSpec{
				// Stuff goes here.
			},
		}
		c.Create(instance)

		time.Sleep(5 * time.Second)

		output := &corev1.ConfigMap{}
		c.Get(helpers.Name("alertmanagerconfig-output"), output)
		config, err := alertconfig.Load(output.Data["alertmanager.yml"])
		Expect(err).ToNot(HaveOccurred())
		Expect(config.Route).ToNot(BeNil())
		Expect(config.Route.Routes).To(HaveLen(1))
	})
})
