/*
Copyright 2018-2019 Ridecell, Inc.

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
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("postgresuser types", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("can create a PostgresUser object", func() {
		c := helpers.Client
		key := types.NamespacedName{
			Name:      "postgresuser",
			Namespace: helpers.Namespace,
		}
		created := &dbv1beta1.PostgresUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "postgresuser",
				Namespace: helpers.Namespace,
			},
			Spec: dbv1beta1.PostgresUserSpec{
				Connection: dbv1beta1.PostgresConnection{
					Host:     "test",
					Port:     5432,
					Username: "test",
					PasswordSecretRef: apihelpers.SecretRef{
						Name: "passwordsecret",
						Key:  "password",
					},
					Database: "test",
				},
				Username: "test",
			},
		}
		err := c.Create(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		fetched := &dbv1beta1.PostgresUser{}
		err = c.Get(context.TODO(), key, fetched)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetched.Spec).To(Equal(created.Spec))
	})
})
