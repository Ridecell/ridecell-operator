/*
Copyright 2020 Ridecell, Inc.

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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"google.golang.org/api/cloudbilling/v1"

	gppcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/gcpproject/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("gcpproject billing Component", func() {
	comp := gppcomponents.NewBilling()
	var billingmock *gppcomponents.GCPCloudBillingMock
	BeforeEach(func() {
		comp = gppcomponents.NewBilling()

		billingmock = &gppcomponents.GCPCloudBillingMock{
			GetProjectBillingInfoFunc: func(_ string) (*cloudbilling.ProjectBillingInfo, error) {
				return &cloudbilling.ProjectBillingInfo{}, nil
			},
			UpdateProjectbillingInfoFunc: func(_ string) (*cloudbilling.ProjectBillingInfo, error) {
				return &cloudbilling.ProjectBillingInfo{}, nil
			},
		}
		comp.InjectBilling(billingmock)
		os.Setenv("GOOGLE_BILLING_ACCOUNT_NAME", "billing-account-name")
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("does nothing if the flag is not set", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(billingmock.GetProjectBillingInfoCalls()).To(HaveLen(0))
		Expect(billingmock.UpdateProjectbillingInfoCalls()).To(HaveLen(0))
	})

	Describe("with flag set", func() {
		BeforeEach(func() {
			trueBool := true
			instance.Spec.EnableBilling = &trueBool
		})

		It("adds billing to an existing project", func() {
			Expect(comp).To(ReconcileContext(ctx))
			Expect(billingmock.GetProjectBillingInfoCalls()).To(HaveLen(1))
			Expect(billingmock.UpdateProjectbillingInfoCalls()).To(HaveLen(1))
		})

		It("does nothing to an account with existing billing account", func() {
			billingmock.GetProjectBillingInfoFunc = func(_ string) (*cloudbilling.ProjectBillingInfo, error) {
				return &cloudbilling.ProjectBillingInfo{BillingAccountName: "billing-account-name"}, nil
			}
			Expect(comp).To(ReconcileContext(ctx))
			Expect(billingmock.GetProjectBillingInfoCalls()).To(HaveLen(1))
			Expect(billingmock.UpdateProjectbillingInfoCalls()).To(HaveLen(0))
		})
	})

})
