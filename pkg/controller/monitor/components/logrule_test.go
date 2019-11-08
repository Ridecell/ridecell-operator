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
	"os"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	mcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/monitor/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_sumologic"
)

var _ = Describe("Monitor Notification Component", func() {
	comp := mcomponents.NewLogrule()
	fake_sumologic.Run()
	BeforeEach(func() {
		os.Setenv("SUMO_MOCK_URL", "http://localhost:8083")
		os.Setenv("ALERTMANAGER_NAME", "dummy")
	})

	It("Is reconcilable", func() {
		instance.Spec.LogAlertRules = []monitoringv1beta1.LogAlertRule{
			monitoringv1beta1.LogAlertRule{
				Name:          "Look for bad things realtime",
				Description:   "looking for bad thing",
				Query:         `_sourceCategory=microservices/prod/us/job-management/* ("cancel_job_if_vehicle_got_reserved_or_moved" AND "TASK_FAILED") OR ("cancel_single_job_vehicle_reserved_task" AND "TASK_FAILED") OR ("cancel_single_job_location_mismatch_task" AND "TASK_FAILED")`,
				Condition:     "gt",
				Threshold:     4,
				Schedule:      "* * 0 0 0 0",
				Range:         "-15m",
				Severity:      "info",
				Runbook:       "https://ridecell.quip.com/ajDsAmRnWFQE/Monitoring",
				ThresholdType: "group",
			},
		}
		instance.Spec.ServiceName = "dev-foo-service"
		Expect(comp).To(ReconcileContext(ctx))
	})
})
