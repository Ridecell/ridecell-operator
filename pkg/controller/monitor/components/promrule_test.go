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

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	mcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/monitor/components"
	pomonitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
)

var _ = Describe("Monitor Promrule Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = mcomponents.NewPromrule()
	})

	It("creates a prometheus rule", func() {
		instance.Spec.MetricAlertRules = []monitoringv1beta1.MetricAlertRule{
			{
				Alert:       "HighErrorRate",
				Expr:        `job:request_latency_seconds:mean5m{job=", "} > 0.5`,
				Labels:      map[string]string{"severity": "page"},
				Annotations: map[string]string{"summary": "High request latency"},
			},
		}

		Expect(comp).To(ReconcileContext(ctx))

		rule := &pomonitoringv1.PrometheusRule{}
		err := ctx.Get(context.Background(), types.NamespacedName{Name: "foo", Namespace: "default"}, rule)
		Expect(err).ToNot(HaveOccurred())
		Expect(rule.Spec.Groups).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules).To(HaveLen(1))
		Expect(rule.Spec.Groups[0].Rules[0].Alert).To(Equal("HighErrorRate"))
	})
})
