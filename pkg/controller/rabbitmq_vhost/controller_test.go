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

package rabbitmq_vhost_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("RabbitmqVhost controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var rabbitmqvhost *dbv1beta1.RabbitmqVhost

	BeforeEach(func() {
		// Check for required environment variables.
		if os.Getenv("RABBITMQ_HOST_DEV") == "" || os.Getenv("RABBITMQ_SUPERUSER") == "" || os.Getenv("RABBITMQ_SUPERUSER_PASSWORD") == "" {
			if os.Getenv("CI") == "" {
				Skip("Skipping RabbitMQ controller tests")
			} else {
				Fail("RabbitMQ test environment not configured")
			}
		}

		helpers = testHelpers.SetupTest()
		rabbitmqvhost = &dbv1beta1.RabbitmqVhost{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
			Spec: dbv1beta1.RabbitmqVhostSpec{
				VhostName: "ridecell-test",
				Connection: dbv1beta1.RabbitmqConnection{
					Production:   false,
					InsecureSkip: false,
				},
			},
		}
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			vhosts := &dbv1beta1.RabbitmqVhostList{}
			err := helpers.Client.List(context.Background(), nil, vhosts)
			if err != nil {
				fmt.Printf("!!!!!! %s\n", err)
			} else {
				fmt.Print("Instances:\n")
				for _, item := range vhosts.Items {
					if item.Namespace == helpers.Namespace {
						fmt.Printf("\t%s %#v\n", item.Name, item.Status)
					}
				}
			}
		}

		helpers.TeardownTest()
	})

	It("Runs a basic reconcile", func() {
		c := helpers.TestClient

		// Connect to RabbitMQ.
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		}
		rmqc, err := rabbithole.NewTLSClient(os.Getenv("RABBITMQ_HOST_DEV"), os.Getenv("RABBITMQ_SUPERUSER"), os.Getenv("RABBITMQ_SUPERUSER_PASSWORD"), transport)
		Expect(err).ToNot(HaveOccurred())

		// Confirm that our credentials work.
		_, err = rmqc.ListVhosts()
		Expect(err).ToNot(HaveOccurred())

		// Create our vhost and wait for it to be Ready.
		c.Create(rabbitmqvhost)
		fetchVhost := &dbv1beta1.RabbitmqVhost{}
		c.EventuallyGet(helpers.Name("test"), fetchVhost, c.EventuallyStatus(dbv1beta1.StatusReady))

		// Check that the vhost exists.
		vhosts, err := rmqc.ListVhosts()
		Expect(err).ToNot(HaveOccurred())
		GetName := func(vhost rabbithole.VhostInfo) string { return vhost.Name }
		Expect(vhosts).To(ContainElement(WithTransform(GetName, Equal("ridecell-test"))))
	})
})
