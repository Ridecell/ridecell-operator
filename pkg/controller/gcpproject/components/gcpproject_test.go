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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/googleapi"

	gppcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/gcpproject/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("serviceaccount serviceaccount Component", func() {
	comp := gppcomponents.NewProject()
	var mock *gppcomponents.GCPCloudResourceManagerMock
	BeforeEach(func() {
		os.Setenv("GOOGLE_ORGANIZATION_ID", "12345")
		comp = gppcomponents.NewProject()
		mock = &gppcomponents.GCPCloudResourceManagerMock{
			CreateFunc: func(_ *components.ComponentContext, _ string) (*cloudresourcemanager.Operation, error) {
				return &cloudresourcemanager.Operation{}, nil
			},
			GetFunc: func(_ string) (*cloudresourcemanager.Project, error) {
				return &cloudresourcemanager.Project{}, nil
			},
			GetOperationFunc: func(_ string) (*cloudresourcemanager.Operation, error) {
				return &cloudresourcemanager.Operation{}, nil
			},
		}
		comp.InjectCRM(mock)
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("does nothing if the project already exists", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mock.CreateCalls()).To(HaveLen(0))
	})

	It("creates the project if it doesn't exist", func() {
		mock.CreateFunc = func(_ *components.ComponentContext, _ string) (*cloudresourcemanager.Operation, error) {
			return &cloudresourcemanager.Operation{Done: false, Name: ""}, nil
		}
		mock.GetFunc = func(_ string) (*cloudresourcemanager.Project, error) {
			return nil, &googleapi.Error{Code: 404}
		}

		res, err := comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter).To(Equal(time.Minute))
		Expect(mock.GetCalls()).To(HaveLen(1))
		Expect(mock.GetOperationCalls()).To(HaveLen(0))
		Expect(mock.CreateCalls()).To(HaveLen(1))

		// Set status to an expected name
		instance.Status.OperationName = "creating-a-project"

		// Run again while operation not done, make sure nothing changes.
		res, err = comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter).To(Equal(time.Minute))
		Expect(mock.GetCalls()).To(HaveLen(2))
		Expect(mock.GetOperationCalls()).To(HaveLen(1))
		Expect(mock.CreateCalls()).To(HaveLen(1))

		// Mark operation as done
		mock.GetOperationFunc = func(operationName string) (*cloudresourcemanager.Operation, error) {
			if operationName == "creating-a-project" {
				return &cloudresourcemanager.Operation{Done: true}, nil
			}
			return nil, &googleapi.Error{Code: 404}
		}
		res, err = comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter).To(Equal(time.Second * 0))
		Expect(mock.GetCalls()).To(HaveLen(3))
		Expect(mock.GetOperationCalls()).To(HaveLen(2))
		Expect(mock.CreateCalls()).To(HaveLen(1))
	})
})
