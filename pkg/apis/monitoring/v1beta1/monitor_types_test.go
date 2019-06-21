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

package v1beta1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("monitor types", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("can create a Monitor object", func() {
		c := helpers.Client
		key := types.NamespacedName{
			Name:      "auth",
			Namespace: helpers.Namespace,
		}
		created := &monitoringv1beta1.Monitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "auth",
				Namespace: helpers.Namespace,
			},
			Spec: monitoringv1beta1.MonitorSpec{
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
				},
			},
		}
		err := c.Create(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		fetched := &monitoringv1beta1.Monitor{}
		err = c.Get(context.TODO(), key, fetched)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetched.Spec).To(Equal(created.Spec))
	})
})
