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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	iam "google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	sacomponents "github.com/Ridecell/ridecell-operator/pkg/controller/serviceaccount/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("serviceaccount key Component", func() {
	comp := sacomponents.NewKey()
	var mock *sacomponents.KeyManagerMock
	BeforeEach(func() {
		comp = sacomponents.NewKey()
		mock = &sacomponents.KeyManagerMock{
			CreateFunc: func(_ string, _ *iam.CreateServiceAccountKeyRequest) (*iam.ServiceAccountKey, error) {
				return &iam.ServiceAccountKey{}, nil
			},
			DeleteFunc: func(_ string) (*iam.Empty, error) {
				return &iam.Empty{}, nil
			},
			ListFunc: func(_ string, _ ...string) (*iam.ListServiceAccountKeysResponse, error) {
				return &iam.ListServiceAccountKeysResponse{}, nil
			},
		}
		comp.InjectKM(mock)
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("does nothing if the key already exists", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "test-user.gcp-credentials", Namespace: "default"},
		}
		ctx.Client = fake.NewFakeClient(instance, secret)
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mock.CreateCalls()).To(HaveLen(0))
	})

	It("creates the key if it doesn't exist", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mock.CreateCalls()).To(HaveLen(1))
	})
})
