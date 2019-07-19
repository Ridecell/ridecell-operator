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

package rabbitmquser_test

import (
	"fmt"
	"os"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

var _ = Describe("RabbitmqUser controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var user *dbv1beta1.RabbitmqUser

	BeforeEach(func() {
		// Check for required environment variables.
		if os.Getenv("RABBITMQ_URI") == "" {
			if os.Getenv("CI") == "" {
				Skip("Skipping RabbitMQ controller tests")
			} else {
				Fail("RabbitMQ test environment not configured")
			}
		}

		helpers = testHelpers.SetupTest()
		user = &dbv1beta1.RabbitmqUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
			Spec: dbv1beta1.RabbitmqUserSpec{
				Tags: "administrator",
			},
		}
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			users := &dbv1beta1.RabbitmqUserList{}
			helpers.TestClient.List(users)
			fmt.Print("Instances:\n")
			for _, item := range users.Items {
				if item.Namespace == helpers.Namespace {
					fmt.Printf("\t%s %#v\n", item.Name, item.Status)
				}
			}
		}

		helpers.TeardownTest()
	})

	It("Runs a basic reconcile", func() {
		c := helpers.TestClient

		// Connect to RabbitMQ.
		rmqc, err := utils.OpenRabbit(nil, nil, utils.RabbitholeClientFactory)
		Expect(err).ToNot(HaveOccurred())

		// Confirm that our credentials work.
		_, err = rmqc.ListVhosts()
		Expect(err).ToNot(HaveOccurred())

		// Create our user.
		c.Create(user)

		// Wait for the user to be ready.
		fetchUser := &dbv1beta1.RabbitmqUser{}
		c.EventuallyGet(helpers.Name("test"), fetchUser, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check that the user exists.
		users, err := rmqc.ListUsers()
		Expect(err).ToNot(HaveOccurred())
		GetName := func(user rabbithole.UserInfo) string { return user.Name }
		Expect(users).To(ContainElement(WithTransform(GetName, Equal("test"))))

		// Try to connect as the user.
		secret := &corev1.Secret{}
		conn := fetchUser.Status.Connection
		c.Get(helpers.Name(conn.PasswordSecretRef.Name), secret)
		password := string(secret.Data[conn.PasswordSecretRef.Key])
		// Hardcoding non-TLS here, probably will break some day.
		userClient, err := rabbithole.NewClient(fmt.Sprintf("http://%v:%v", conn.Host, conn.Port), conn.Username, password)
		Expect(err).ToNot(HaveOccurred())
		_, err = userClient.Overview()
		Expect(err).ToNot(HaveOccurred())
	})
})
