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
	"strings"
	"time"

	// Some slack methods deprecated Nov 25, 2020. Will need to update tests!
	// See nlopes/slack readme and slack api:
	// https://api.slack.com/changelog/2020-01-deprecating-antecedents-to-the-conversations-api
	"github.com/nlopes/slack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ghttp "github.com/onsi/gomega/ghttp"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

// NOTE: To deal with flakey Slack notification history check failures resulting from concurrent test
// runs, use testRunId in the instance name.
var _ = Describe("Summon controller notifications", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *summonv1beta1.SummonPlatform
	testRunId, err := utils.RandomString(6)
	if err != nil {
		Fail("Unable to setup testrun id")
	}

	// Helper function for slack notification test cases to retrieve only test case related slack history.
	getTestRelevantHistory := func(testIdentity string, slackClient *slack.Client, slackChannel string, lastTimestamp string) slack.History {
		historyParams := slack.NewHistoryParameters()
		historyParams.Oldest = lastTimestamp
		history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
		Expect(err).ToNot(HaveOccurred())
		filteredMsgs := []slack.Message{}
		// Get messages pertaining only to testIdentity.
		for _, msg := range history.Messages {
			if strings.Contains(msg.Attachments[0].Text, testIdentity) {
				filteredMsgs = append(filteredMsgs, msg)
			}
		}
		// Replace the history messages with the filtered one.
		history.Messages = filteredMsgs
		return *history
	}

	// Helper function to provide more insight into what slack messages were posted if the number of messages
	// seen were not expected.
	describeMsgs := func(msgs []slack.Message) string {
		description := "Messages Seen:\n"
		for _, msg := range msgs {
			description += msg.Attachments[0].Text + "\n"
		}
		return description
	}

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
				"filler":      []byte{},
				"FERNET_KEYS": []byte("myfernetkey1")}}
		helpers.TestClient.Create(appSecrets)

		// Set up the instance object for other tests.
		instance = &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "notifytest", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "80813-eb6b515-master",
				Secrets: []string{"testsecret"},
			},
		}

		// Increase the default EventuallyGet timeout.
		test_helpers.DefaultTimeout = 60 * time.Second
	})

	AfterEach(func() {
		// Restore the default EventuallyGet timeout.
		test_helpers.DefaultTimeout = 30 * time.Second

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

		// Check that a migration Job was created.
		job := &batchv1.Job{}
		c.EventuallyGet(helpers.Name(name+"-migrations"), job)

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

		// Mark the deployments as ready.
		updateDeployment := func(s string, count int32) {
			deployment := &appsv1.Deployment{}
			c.EventuallyGet(helpers.Name(name+"-"+s), deployment)
			deployment.Status.Replicas = count
			deployment.Status.ReadyReplicas = count
			deployment.Status.AvailableReplicas = count
			//The below conditions are checked by ENABLE_NEW_STATUS.
			// (AvailableReplicas not checked under ENABLE_NEW_STATUS)
			deployment.Status.UnavailableReplicas = 0
			deployment.Status.UpdatedReplicas = count
			c.Status().Update(deployment)
		}
		updateDeployment("web", 1)
		updateDeployment("daphne", 1)
		updateDeployment("celeryd", 1)
		updateDeployment("channelworker", 1)
		updateDeployment("static", 1)
		updateDeployment("dispatch", 0)
		updateDeployment("businessportal", 0)
		updateDeployment("hwaux", 0)
		updateDeployment("tripshare", 0)

		// Mark the statefulset as ready.
		statefulset := &appsv1.StatefulSet{}
		c.EventuallyGet(helpers.Name(name+"-celerybeat"), statefulset)
		statefulset.Status.Replicas = 1
		statefulset.Status.ReadyReplicas = 1
		statefulset.Status.UpdatedReplicas = 1
		c.Status().Update(statefulset)
	}

	Context("for Slack", func() {
		var slackClient *slack.Client
		var lastMessage slack.Message
		var testIdentity string

		// The ID of the private group to send to.
		slackChannel := "CKEV56KKJ" // #rcoperator-test. Should only be used by circleci.
		//slackChannel := "CKBMB2E3V" // #rcoperator-test2. Use this one for local testing!

		BeforeEach(func() {
			// Check for Slack API key. If not present, don't run these tests.
			// Allows for easier devX, only need to install the credentials if you are
			// debugging these tests or whatever. Test slack api key available in lastpass
			// under Slack RCOperatorPseudoBot.
			if os.Getenv("SLACK_API_KEY") == "" {
				Skip("$SLACK_API_KEY not set, skipping Slack tests")
			}

			os.Setenv("ENABLE_NEW_STATUS_CHECK", "true")

			// Set up Slack client with the test user credentials and find the most recent message.
			slackClient = slack.New(os.Getenv("SLACK_API_KEY"))
			historyParams := slack.NewHistoryParameters()
			historyParams.Count = 1
			history, err := slackClient.GetChannelHistory(slackChannel, historyParams)
			Expect(err).ToNot(HaveOccurred())
			lastMessage = history.Messages[0]

			// Configure the instance.
			instance.Spec.Notifications.SlackChannel = slackChannel

			// Unique test case id to help filter out irrelevant teardown error notifications.
			testCaseId, err := utils.RandomString(3)
			if err != nil {
				Fail("Unable to setup testCaseId")
			}

			testIdentity = testRunId + "-" + testCaseId
		})

		Context("Summon Platform", func() {
			It("sends a single success notification on deploy", func() {
				c := helpers.TestClient

				// Advance all the various things.
				deployInstance(testIdentity + "-notifytest")

				// Check that things are ready.
				fetchInstance := &summonv1beta1.SummonPlatform{}
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Check that the notification state saved correctly. This is mostly to wait until the final reconcile before exiting the test.
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyValue(Equal("80813-eb6b515-master"), func(obj runtime.Object) (interface{}, error) {
					return obj.(*summonv1beta1.SummonPlatform).Status.Notification.SummonVersion, nil
				}))

				// Find all messages since the start of the test.
				history := getTestRelevantHistory(testIdentity, slackClient, slackChannel, lastMessage.Timestamp)
				Expect(len(history.Messages)).To(Equal(1), describeMsgs(history.Messages))
				Expect(history.Messages[0].Attachments).To(HaveLen(1))
				Expect(history.Messages[0].Attachments[0].Color).To(Equal("2eb886"))
			})

			It("sends a single success notification on deploy, even with subsequent reconciles", func() {
				c := helpers.TestClient

				// Advance all the various things.
				deployInstance(testIdentity + "-notifytest")

				// Check that things are ready.
				fetchInstance := &summonv1beta1.SummonPlatform{}
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Simulate a pod delete.
				deployment := &appsv1.Deployment{}
				c.Get(helpers.Name(testIdentity+"-notifytest-web"), deployment)
				deployment.Status.ReadyReplicas = 0
				deployment.Status.UpdatedReplicas = 0
				deployment.Status.AvailableReplicas = 0
				c.Status().Update(deployment)
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusDeploying))
				deployment.Status.ReadyReplicas = 1
				deployment.Status.UpdatedReplicas = 1
				deployment.Status.AvailableReplicas = 1
				c.Status().Update(deployment)
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Find all messages since the start of the test.
				history := getTestRelevantHistory(testIdentity, slackClient, slackChannel, lastMessage.Timestamp)
				Expect(len(history.Messages)).To(Equal(1), describeMsgs(history.Messages))
			})

			It("sends two success notifications for two different clusters", func() {
				c := helpers.TestClient

				// Advance all the various things.
				deployInstance(testIdentity + "-notifytest")
				deployInstance(testIdentity + "-notifytest2")

				// Check that things are ready.
				fetchInstance := &summonv1beta1.SummonPlatform{}
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Find all messages since the start of the test.
				history := getTestRelevantHistory(testIdentity, slackClient, slackChannel, lastMessage.Timestamp)
				Expect(len(history.Messages)).To(Equal(2), describeMsgs(history.Messages))

			})

			It("sends a single error notification on something going wrong", func() {
				c := helpers.TestClient
				instance.Name = testIdentity + "-notifytest"
				// Create the SummonPlatform.
				c.Create(instance)

				// Simulate a Postgres error.
				postgres := &dbv1beta1.PostgresDatabase{}
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), postgres)
				postgres.Status.Status = dbv1beta1.StatusError
				postgres.Status.Message = "Simulated DB error"
				c.Status().Update(postgres)

				// Wait.
				fetchInstance := &summonv1beta1.SummonPlatform{}
				c.EventuallyGet(helpers.Name(testIdentity+"-notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusError))

				// Check that exactly one message happened
				history := getTestRelevantHistory(testIdentity, slackClient, slackChannel, lastMessage.Timestamp)
				Expect(len(history.Messages)).To(Equal(1), describeMsgs(history.Messages))
				Expect(history.Messages[0].Attachments).To(HaveLen(1))
				Expect(history.Messages[0].Attachments[0].Color).To(Equal("a30200"))
			})
		})

		Context("Components", func() {
			It("sends a success notification for summonplatform and one for businessportal on deploy", func() {
				c := helpers.TestClient

				// Include BusinessPortal in deploy.
				instance.Spec.BusinessPortal.Version = "123-abc123-businessportal"
				// Create instance.
				deployInstance(testIdentity + "-componentnotifytest")
				// handle businessportal deployment
				deployment := &appsv1.Deployment{}
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest-businessportal"), deployment)
				deployment.Status.Replicas = 1
				deployment.Status.ReadyReplicas = 1
				deployment.Status.UpdatedReplicas = 1
				c.Status().Update(deployment)
				// For ENABLE_NEW_STATUS_CHECK
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest-dispatch"), deployment)

				deployment.Status.Replicas = 0
				deployment.Status.ReadyReplicas = 0
				deployment.Status.UpdatedReplicas = 0
				c.Status().Update(deployment)
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest-tripshare"), deployment)

				deployment.Status.Replicas = 0
				deployment.Status.ReadyReplicas = 0
				deployment.Status.UpdatedReplicas = 0
				c.Status().Update(deployment)
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest-hwaux"), deployment)
				deployment.Status.Replicas = 0
				deployment.Status.ReadyReplicas = 0
				deployment.Status.UpdatedReplicas = 0
				c.Status().Update(deployment)

				// Check that components are ready.
				fetchInstance := &summonv1beta1.SummonPlatform{}
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Check the notification statuses. This is mostly to wait until the final reconcile before exiting the test.
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest"), fetchInstance, c.EventuallyValue(Equal("80813-eb6b515-master"), func(obj runtime.Object) (interface{}, error) {
					return obj.(*summonv1beta1.SummonPlatform).Status.Notification.SummonVersion, nil
				}))
				c.EventuallyGet(helpers.Name(testIdentity+"-componentnotifytest"), fetchInstance, c.EventuallyValue(Equal("123-abc123-businessportal"), func(obj runtime.Object) (interface{}, error) {
					return obj.(*summonv1beta1.SummonPlatform).Status.Notification.BusinessPortalVersion, nil
				}))

				// Find all messages since the start of the test.
				history := getTestRelevantHistory(testIdentity, slackClient, slackChannel, lastMessage.Timestamp)
				Expect(len(history.Messages)).To(Equal(2), describeMsgs(history.Messages))
				Expect(history.Messages[0].Attachments).To(HaveLen(1))
				Expect(history.Messages[0].Attachments[0].Color).To(Equal("2eb886"))
				Expect(history.Messages[0].Attachments[0].Title).To(Equal(testIdentity + "-componentnotifytest.ridecell.us comp-business-portal Deployment"))
				Expect(history.Messages[1].Attachments).To(HaveLen(1))
				Expect(history.Messages[1].Attachments[0].Color).To(Equal("2eb886"))
				Expect(history.Messages[1].Attachments[0].Title).To(Equal(testIdentity + "-componentnotifytest.ridecell.us summon-platform Deployment"))
			})

			It("sends a single success notification per unique deploy, even with subsequent reconciles", func() {
				c := helpers.TestClient

				// Include dispatch and hwaux in deploy.
				instance.Spec.Dispatch.Version = "75-63a9598-master"
				instance.Spec.HwAux.Version = "25-ccb55f7-master"
				// Create instance.
				deployInstance(testIdentity + "-summon-dispatch-hwaux-test")

				// handle dispatch and hwaux deployment
				deployment := &appsv1.Deployment{}
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test-dispatch"), deployment)
				deployment.Status.Replicas = 1
				deployment.Status.ReadyReplicas = 1
				deployment.Status.UpdatedReplicas = 1
				c.Status().Update(deployment)
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test-hwaux"), deployment)
				deployment.Status.Replicas = 1
				deployment.Status.ReadyReplicas = 1
				deployment.Status.UpdatedReplicas = 1
				c.Status().Update(deployment)

				// Check that things are ready.
				fetchInstance := &summonv1beta1.SummonPlatform{}
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Check the notification status updated.
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test"), fetchInstance, c.EventuallyValue(Equal("80813-eb6b515-master"), func(obj runtime.Object) (interface{}, error) {
					return obj.(*summonv1beta1.SummonPlatform).Status.Notification.SummonVersion, nil
				}))

				// SummonPlatform version is changing, but not its components.
				fetchInstance.Spec.Version = "456-abababc-master"
				c.Update(fetchInstance)
				// Simulate old web deployment getting replaced with the new version.
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test-web"), deployment)
				deployment.Status.ReadyReplicas = 0
				deployment.Status.UpdatedReplicas = 0
				deployment.Status.AvailableReplicas = 0
				c.Status().Update(deployment)

				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusMigrating))
				job := &batchv1.Job{}
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test-migrations"), job)
				job.Status.Succeeded = 1
				c.Status().Update(job)
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusDeploying))
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test-web"), deployment)
				deployment.Status.ReadyReplicas = 1
				deployment.Status.UpdatedReplicas = 1
				deployment.Status.AvailableReplicas = 1
				c.Status().Update(deployment)
				c.EventuallyGet(helpers.Name(testIdentity+"-summon-dispatch-hwaux-test"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

				// Find all messages since the start of the test.
				history := getTestRelevantHistory(testIdentity, slackClient, slackChannel, lastMessage.Timestamp)
				Expect(len(history.Messages)).To(Equal(4), describeMsgs(history.Messages))
				// One for platform one for dispatch, one for hwaux, and a new one for summon platform version change.
				Expect(history.Messages[0].Attachments).To(HaveLen(1))
				Expect(history.Messages[0].Attachments[0].Color).To(Equal("2eb886"))
				Expect(history.Messages[0].Attachments[0].Title).To(Equal(testIdentity + "-summon-dispatch-hwaux-test.ridecell.us summon-platform Deployment"))
				Expect(history.Messages[1].Attachments).To(HaveLen(1))
				Expect(history.Messages[1].Attachments[0].Color).To(Equal("2eb886"))
				Expect(history.Messages[1].Attachments[0].Title).To(Equal(testIdentity + "-summon-dispatch-hwaux-test.ridecell.us comp-hw-aux Deployment"))
				Expect(history.Messages[2].Attachments).To(HaveLen(1))
				Expect(history.Messages[2].Attachments[0].Color).To(Equal("2eb886"))
				Expect(history.Messages[2].Attachments[0].Title).To(Equal(testIdentity + "-summon-dispatch-hwaux-test.ridecell.us comp-dispatch Deployment"))
				Expect(history.Messages[3].Attachments).To(HaveLen(1))
				Expect(history.Messages[3].Attachments[0].Color).To(Equal("2eb886"))
				Expect(history.Messages[3].Attachments[0].Title).To(Equal(testIdentity + "-summon-dispatch-hwaux-test.ridecell.us summon-platform Deployment"))
			})
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

		It("sends a single post request per component deployed", func() {
			c := helpers.TestClient

			// Set up verification handler to check our request body.
			deployStatusServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest", "deploy_user": "ridecell-operator","environment": "%s","tag": "80813-eb6b515-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest comp-trip-share", "deploy_user": "ridecell-operator","environment": "%s","tag": "129-c09a54b-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			instance.Spec.TripShare.Version = "129-c09a54b-master"
			// Advance all the various things.
			deployInstance("notifytest")
			deployment := &appsv1.Deployment{}
			c.EventuallyGet(helpers.Name("notifytest-tripshare"), deployment)
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			deployment.Status.UpdatedReplicas = 1
			c.Status().Update(deployment)

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			// Check that the notification state saved correctly. This is mostly to wait until the final reconcile before exiting the test.
			c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyValue(Equal("80813-eb6b515-master"), func(obj runtime.Object) (interface{}, error) {
				return obj.(*summonv1beta1.SummonPlatform).Status.Notification.SummonVersion, nil
			}))

			// Check post request was actually made to deployment status tool.
			Expect(deployStatusServer.ReceivedRequests()).Should(HaveLen(2))
		})

		It("sends a single post request per unique deploy, even with subsequent reconciles", func() {
			c := helpers.TestClient

			// Set up verification handler to check our request body.
			deployStatusServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest2", "deploy_user": "ridecell-operator","environment": "%s","tag": "80813-eb6b515-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest2 comp-business-portal", "deploy_user": "ridecell-operator","environment": "%s","tag": "157-2a74b0f-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest2 comp-trip-share", "deploy_user": "ridecell-operator","environment": "%s","tag": "129-c09a54b-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/"),
					ghttp.VerifyJSON(fmt.Sprintf(`{"customer_name": "notifytest2 comp-business-portal", "deploy_user": "ridecell-operator","environment": "%s","tag": "154-b132c58-master"}`, helpers.Namespace)),
					ghttp.RespondWith(http.StatusOK, ""),
				),
			)

			instance.Spec.TripShare.Version = "129-c09a54b-master"
			instance.Spec.BusinessPortal.Version = "157-2a74b0f-master"
			// Advance all the various things.
			deployInstance("notifytest2")
			deployment := &appsv1.Deployment{}
			c.EventuallyGet(helpers.Name("notifytest2-tripshare"), deployment)
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			deployment.Status.UpdatedReplicas = 1
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest2-businessportal"), deployment)
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			deployment.Status.UpdatedReplicas = 1
			c.Status().Update(deployment)

			// Check that things are ready.
			fetchInstance := &summonv1beta1.SummonPlatform{}
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

			fetchInstance.Spec.BusinessPortal.Version = "154-b132c58-master"
			c.Update(fetchInstance)
			// Simulate a pod delete.
			deployment = &appsv1.Deployment{}
			c.Get(helpers.Name("notifytest2-web"), deployment)
			deployment.Status.ReadyReplicas = 0
			deployment.Status.AvailableReplicas = 0
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusDeploying))
			deployment.Status.ReadyReplicas = 1
			deployment.Status.AvailableReplicas = 1
			c.Status().Update(deployment)
			c.EventuallyGet(helpers.Name("notifytest2"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))
			Expect(fetchInstance.Status.Notification.BusinessPortalVersion).To(Equal("154-b132c58-master"))
			// Expect single deployment status post request.
			Expect(deployStatusServer.ReceivedRequests()).Should(HaveLen(4))
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
