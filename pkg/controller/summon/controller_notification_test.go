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

package summon_test

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/nlopes/slack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ghttp "github.com/onsi/gomega/ghttp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("Summon controller notifications", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *summonv1beta1.SummonPlatform

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		pullSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "pull-secret", Namespace: helpers.OperatorNamespace},
			Type:       "kubernetes.io/dockerconfigjson",
			StringData: map[string]string{".dockerconfigjson": "{\"auths\": {}}"}}
		helpers.TestClient.Create(pullSecret)

		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"filler": []byte{}}}
		helpers.TestClient.Create(appSecrets)

		// Set up the instance object for other tests.
		instance = &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "notifytest", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "80813-eb6b515-master",
				Secrets: []string{"testsecret"},
			},
		}
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			summons := &summonv1beta1.SummonPlatformList{}
			err := helpers.Client.List(context.Background(), nil, summons)
			if err != nil {
				fmt.Printf("!!!!!! %s\n", err)
			} else {
				fmt.Print("Failed instances:\n")
				for _, item := range summons.Items {
					if item.Namespace == helpers.Namespace {
						fmt.Printf("\t%s %#v\n", item.Name, item.Status)
					}
				}
			}
		}
		helpers.TeardownTest()
	})

	deployInstance := func(name string) {
		c := helpers.TestClient

		// Create the SummonPlatform.
		instance.Name = name
		instance.ResourceVersion = ""
		c.Create(instance)

		// Mark the PullSecret as ready.
		pullsecret := &secretsv1beta1.PullSecret{}
		c.EventuallyGet(helpers.Name(name+"-pullsecret"), pullsecret)
		pullsecret.Status.Status = secretsv1beta1.StatusReady
		c.Status().Update(pullsecret)

		// Create the AWS credentials for app secrets because the IAMUser controller isn't running.
		awsSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name + ".aws-credentials", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIAtest",
				"AWS_SECRET_ACCESS_KEY": "test",
			},
		}
		c.Create(awsSecret)

		// Wait for the database to be created.
		db := &dbv1beta1.PostgresDatabase{}
		c.EventuallyGet(helpers.Name(name), db)

		// Create a fake Postgres credentials secret.
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name + ".postgres-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretdbpass",
			},
		}
		c.Create(dbSecret)

		// Set the status of the DB to ready.
		db.Status.Status = dbv1beta1.StatusReady
		db.Status.Connection = dbv1beta1.PostgresConnection{
			Host:     name,
			Username: "summon",
			Database: "summon",
			PasswordSecretRef: apihelpers.SecretRef{
				Name: dbSecret.Name,
				Key:  "password",
			},
		}
		c.Status().Update(db)

		// Set up RabbitMQ.
		rmqSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name + ".rabbitmq-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretrabbitpass",
			},
		}
		c.Create(rmqSecret)
		rmqVhost := &dbv1beta1.RabbitmqVhost{}
		c.EventuallyGet(helpers.Name(name), rmqVhost)
		rmqVhost.Status = dbv1beta1.RabbitmqVhostStatus{
			Status: dbv1beta1.StatusReady,
			Connection: dbv1beta1.RabbitmqStatusConnection{
				Host:     "rabbitmqserver",
				Username: name + "-user",
				Vhost:    name,
				PasswordSecretRef: apihelpers.SecretRef{
					Name: name + ".rabbitmq-user-password",
					Key:  "password",
				},
			},
		}
		c.Status().Update(rmqVhost)

		// Check that a migration object was created.
		migration := &dbv1beta1.MigrationJob{}
		c.EventuallyGet(helpers.Name(name), migration)

		// Mark the migrations as successful.
		migration.Status.Status = dbv1beta1.StatusReady
		c.Status().Update(migration)

		// Mark the deployments as ready.
		updateDeployment := func(s string) {
			deployment := &appsv1.Deployment{}
			c.EventuallyGet(helpers.Name(name+"-"+s), deployment)
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			deployment.Status.AvailableReplicas = 1
			c.Status().Update(deployment)
		}
		updateDeployment("web")
		updateDeployment("daphne")
		updateDeployment("celeryd")
		updateDeployment("channelworker")
		updateDeployment("static")

		// Mark the statefulset as ready.
		statefulset := &appsv1.StatefulSet{}
		c.EventuallyGet(helpers.Name(name+"-celerybeat"), statefulset)
		statefulset.Status.Replicas = 1
		statefulset.Status.ReadyReplicas = 1
		c.Status().Update(statefulset)
	}

	Context("for Slack", func() {
		var slackClient *slack.Client
		var lastMessage slack.Message

		// The ID of the private group to send to.
		slackChannel := "CKEV56KKJ" // #rcoperator-test

		BeforeEach(func() {
			// Check for Slack API key. If not present, don't run these tests.
			// Allows for easier devX, only need to install the credentials if you are
			// debugging these tests or whatever. Test slack api key available in lastpass
			// under Slack RCOperatorPseudoBot.
			if os.Getenv("SLACK_API_KEY") == "" {
				Skip("$SLACK_API_KEY not set, skipping Slack tests")
			}

			// Set up Slack client with the test user credentials and find the most recent message.
			slackClient = slack.New(os.Getenv("SLACK_API_KEY"))
			historyParams := slack.NewHistoryParameters()
			historyParams.Count = 1
			history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
			Expect(err).ToNot(HaveOccurred())
			lastMessage = history.Messages[0]

			// Configure the instance.
			instance.Spec.Notifications.SlackChannel = slackChannel
		})

		It("sends a single success notification on deploy", func() {
			c := helpers.TestClient

			// Advance all the various things.
			deployInstance("notifytest")

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Check that the notification state saved correctly. This is mostly to wait until the final reconcile before exiting the test.
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyValue(Equal("80813-eb6b515-master"), func(obj runtime.Object) (interface{}, error) {
				return obj.(*summonv1beta1.SummonPlatform).Status.Notification.NotifyVersion, nil
			}))

			// Find all messages since the start of the test.
			historyParams := slack.NewHistoryParameters()
			historyParams.Oldest = lastMessage.Timestamp
			history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
			Expect(err).ToNot(HaveOccurred())
			Expect(history.Messages).To(HaveLen(1))
			Expect(history.Messages[0].Attachments).To(HaveLen(1))
			Expect(history.Messages[0].Attachments[0].Color).To(Equal("2eb886"))
		})

		It("sends a single success notification on deploy, even with subsequent reconciles", func() {
			c := helpers.TestClient

			// Advance all the various things.
			deployInstance("notifytest")

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Simulate a pod delete.
			deployment := &appsv1.Deployment{}
			c.Get(helpers.Name("notifytest-web"), deployment)
			deployment.Status.ReadyReplicas = 0
			deployment.Status.AvailableReplicas = 0
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusDeploying))
			deployment.Status.ReadyReplicas = 1
			deployment.Status.AvailableReplicas = 1
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Find all messages since the start of the test.
			historyParams := slack.NewHistoryParameters()
			historyParams.Oldest = lastMessage.Timestamp
			history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
			Expect(err).ToNot(HaveOccurred())
			Expect(history.Messages).To(HaveLen(1))
		})

		It("sends two success notifications for two different clusters", func() {
			c := helpers.TestClient

			// Advance all the various things.
			deployInstance("notifytest")
			deployInstance("notifytest2")

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Find all messages since the start of the test.
			historyParams := slack.NewHistoryParameters()
			historyParams.Oldest = lastMessage.Timestamp
			history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
			Expect(err).ToNot(HaveOccurred())
			Expect(history.Messages).To(HaveLen(2))
		})

		It("sends a single error notification on something going wrong", func() {
			c := helpers.TestClient

			// Create the SummonPlatform.
			c.Create(instance)

			// Simulate a Postgres error.
			postgres := &dbv1beta1.PostgresDatabase{}
			c.EventuallyGet(helpers.Name("notifytest"), postgres)
			postgres.Status.Status = dbv1beta1.StatusError
			postgres.Status.Message = "Simulated DB error"
			c.Status().Update(postgres)

			// Wait.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusError))

			// Check that exactly one message happened
			historyParams := slack.NewHistoryParameters()
			historyParams.Oldest = lastMessage.Timestamp
			history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
			Expect(err).ToNot(HaveOccurred())
			Expect(history.Messages).To(HaveLen(1))
			Expect(history.Messages[0].Attachments).To(HaveLen(1))
			Expect(history.Messages[0].Attachments[0].Color).To(Equal("a30200"))
		})
	})

	Context("for deployment-status", func() {
		var deployStatusServer *ghttp.Server

		BeforeEach(func() {
			deployStatusServer = ghttp.NewServer()
			instance.Spec.Notifications.DeploymentStatusUrl = deployStatusServer.URL()
		})

		AfterEach(func() {
			deployStatusServer.Close()
		})

		It("sends a single post request on deploy", func() {
			c := helpers.TestClient

			// Set up verification handler to check our request body.
			deployStatusServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest", "deploy_user": "ridecell-operator","environment": "%s","tag": "80813-eb6b515-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			// Advance all the various things.
			deployInstance("notifytest")

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Check that the notification state saved correctly. This is mostly to wait until the final reconcile before exiting the test.
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyValue(Equal("80813-eb6b515-master"), func(obj runtime.Object) (interface{}, error) {
				return obj.(*summonv1beta1.SummonPlatform).Status.Notification.NotifyVersion, nil
			}))

			// Check post request was actually made to deployment status tool.
			Expect(deployStatusServer.ReceivedRequests()).Should(HaveLen(1))
		})

		It("sends a single post request on deploy, even with subsequent reconciles", func() {
			c := helpers.TestClient

			// Set up verification handler to check our request body.
			deployStatusServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest2", "deploy_user": "ridecell-operator","environment": "%s","tag": "80813-eb6b515-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			// Advance all the various things.
			deployInstance("notifytest2")

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Simulate a pod delete.
			deployment := &appsv1.Deployment{}
			c.Get(helpers.Name("notifytest2-web"), deployment)
			deployment.Status.ReadyReplicas = 0
			deployment.Status.AvailableReplicas = 0
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusDeploying))
			deployment.Status.ReadyReplicas = 1
			deployment.Status.AvailableReplicas = 1
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Expect single deployment status post request.
			Expect(deployStatusServer.ReceivedRequests()).Should(HaveLen(1))
		})

		It("sends two post requests for two deploys, each in different clusters", func() {
			c := helpers.TestClient

			// Set up verification handler to check our request body.
			deployStatusServer.AppendHandlers(
				// Verifications for first request (notifytest).
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest", "deploy_user": "ridecell-operator","environment": "%s","tag": "80813-eb6b515-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				// Verifications for second request (notifytest2).
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest2", "deploy_user": "ridecell-operator","environment": "%s","tag": "80813-eb6b515-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			// Advance all the various things.
			deployInstance("notifytest")
			deployInstance("notifytest2")

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Expect two deployment status post request.
			Expect(deployStatusServer.ReceivedRequests()).Should(HaveLen(2))
		})

		It("does not send post request for deploy when something went wrong", func() {
			c := helpers.TestClient

			// Create the SummonPlatform.
			c.Create(instance)

			// Simulate a Postgres error.
			postgres := &dbv1beta1.PostgresDatabase{}
			c.EventuallyGet(helpers.Name("notifytest"), postgres)
			postgres.Status.Status = dbv1beta1.StatusError
			postgres.Status.Message = "It go boom"
			c.Status().Update(postgres)

			// Wait.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusError))

			// Check that exactly deployment status post occurred.
			Expect(deployStatusServer.ReceivedRequests()).Should(HaveLen(0))
		})
	})
})
