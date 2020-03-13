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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"google.golang.org/api/firebase/v1beta1"
	"google.golang.org/api/googleapi"

	gppcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/gcpproject/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("gcpproject firebaseproject Component", func() {
	comp := gppcomponents.NewFirebaseProject()
	var firebasemock *gppcomponents.GCPFirebaseMock
	BeforeEach(func() {
		comp = gppcomponents.NewFirebaseProject()

		firebasemock = &gppcomponents.GCPFirebaseMock{
			GetFunc: func(_ string) (*firebase.FirebaseProject, error) {
				return &firebase.FirebaseProject{}, nil
			},
			GetOperationFunc: func(_ string) (*firebase.Operation, error) {
				return &firebase.Operation{}, nil
			},
			AddFirebaseFunc: func(_ string) (*firebase.Operation, error) {
				return &firebase.Operation{}, nil
			},
		}
		comp.InjectFirebase(firebasemock)
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("does nothing if the flag is not set", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(firebasemock.GetCalls()).To(HaveLen(0))
		Expect(firebasemock.GetOperationCalls()).To(HaveLen(0))
		Expect(firebasemock.AddFirebaseCalls()).To(HaveLen(0))
	})

	It("adds firebase to an existing project", func() {
		trueBool := true
		instance.Spec.EnableFirebase = &trueBool

		firebasemock.GetFunc = func(_ string) (*firebase.FirebaseProject, error) {
			// Return a 404 to signal that firebase has not been added to project
			return nil, &googleapi.Error{Code: 404}
		}

		res, err := comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter).To(Equal(time.Minute))
		Expect(firebasemock.GetCalls()).To(HaveLen(1))
		Expect(firebasemock.GetOperationCalls()).To(HaveLen(0))
		Expect(firebasemock.AddFirebaseCalls()).To(HaveLen(1))

		// Signal firebase operation as done
		instance.Status.FirebaseOperationName = "firebase-operation"
		firebasemock.GetOperationFunc = func(_ string) (*firebase.Operation, error) {
			return &firebase.Operation{Done: true}, nil
		}

		res, err = comp.Reconcile(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter).To(Equal(time.Second * 0))
		Expect(firebasemock.GetCalls()).To(HaveLen(2))
		Expect(firebasemock.GetOperationCalls()).To(HaveLen(1))
		Expect(firebasemock.AddFirebaseCalls()).To(HaveLen(1))
	})
})
