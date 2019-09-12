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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"google.golang.org/api/googleapi"
	iam "google.golang.org/api/iam/v1"

	sacomponents "github.com/Ridecell/ridecell-operator/pkg/controller/serviceaccount/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("serviceaccount serviceaccount Component", func() {
	comp := sacomponents.NewServiceAccount()
	var mock *sacomponents.ServiceAccountManagerMock
	BeforeEach(func() {
		comp = sacomponents.NewServiceAccount()
		mock = &sacomponents.ServiceAccountManagerMock{
			CreateFunc: func(_ string, _ *iam.CreateServiceAccountRequest) (*iam.ServiceAccount, error) {
				return &iam.ServiceAccount{}, nil
			},
			GetFunc: func(_ string) (*iam.ServiceAccount, error) {
				return &iam.ServiceAccount{}, nil
			},
		}
		comp.InjectSAM(mock)
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("does nothing if the account already exists", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mock.CreateCalls()).To(HaveLen(0))
	})

	It("creates the account if it doesn't exist", func() {
		mock.GetFunc = func(_ string) (*iam.ServiceAccount, error) {
			return nil, &googleapi.Error{Code: 404}
		}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mock.CreateCalls()).To(HaveLen(1))
	})
})
