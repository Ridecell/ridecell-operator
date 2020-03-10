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

package v1beta1_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GCPProject types", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("can create a GCPProject object", func() {
		c := helpers.Client
		key := types.NamespacedName{
			Name:      "gcpproject",
			Namespace: helpers.Namespace,
		}
		created := &gcpv1beta1.GCPProject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gcpproject",
				Namespace: helpers.Namespace,
			},
		}
		err := c.Create(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		fetched := &gcpv1beta1.GCPProject{}
		err = c.Get(context.TODO(), key, fetched)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetched.Spec).To(Equal(created.Spec))
	})
})