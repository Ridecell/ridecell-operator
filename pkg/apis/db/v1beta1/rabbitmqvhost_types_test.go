/*
Copyright 2019-2020 Ridecell, Inc.

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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("RabbitmqVhost types", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("can create a RabbitmqVhost object", func() {
		c := helpers.Client
		key := types.NamespacedName{
			Name:      "rabbitmqvhost",
			Namespace: helpers.Namespace,
		}
		created := &dbv1beta1.RabbitmqVhost{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rabbitmqvhost",
				Namespace: helpers.Namespace,
			},
			Spec: dbv1beta1.RabbitmqVhostSpec{
				VhostName: "rabbitmq-test",
				SkipUser:  false,
				Policies: map[string]dbv1beta1.RabbitmqPolicy{
					"rabbitmq-test-p1": dbv1beta1.RabbitmqPolicy{
						Pattern:    "^amq\\.",
						ApplyTo:    "queues",
						Priority:   1,
						Definition: "federation-upstream-set:all;ha-mode:all",
					},
				},
			},
		}
		err := c.Create(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		fetched := &dbv1beta1.RabbitmqVhost{}
		err = c.Get(context.TODO(), key, fetched)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetched.Spec).To(Equal(created.Spec))
	})
})
