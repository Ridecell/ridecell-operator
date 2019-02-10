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
	"os"

	"github.com/nlopes/slack"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("Summon controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *summonv1beta1.SummonPlatform
	var slackClient *slack.Client
	var lastMessage slack.Message

	// The ID of the private group to send to.
	slackChannel := "GFLUB5A49"

	BeforeEach(func() {
		// Check for both Slack API tokens. If not present, don't run these tests.
		// Allows for easier devX, only need to install the credentials if you are
		// debugging these tests or whatever.
		if os.Getenv("SLACK_API_KEY") == "" {
			Skip("$SLACK_API_KEY not set, skipping Slack tests")
		}
		if os.Getenv("SLACK_API_KEY_TESTUSER") == "" {
			Skip("$SLACK_API_KEY_TESTUSER not set, skipping Slack tests")
		}

		// Set up Slack client with the test user credentials and find the most recent message.
		slackClient = slack.New(os.Getenv("SLACK_API_KEY_TESTUSER"))
		historyParams := slack.NewHistoryParameters()
		historyParams.Count = 1
		history, err := slackClient.GetGroupHistory(slackChannel, historyParams)
		Expect(err).ToNot(HaveOccurred())
		lastMessage = history.Messages[0]

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
				Database: summonv1beta1.DatabaseSpec{
					ExclusiveDatabase: true,
				},
				Notifications: summonv1beta1.NotificationsSpec{
					SlackChannel: slackChannel,
				},
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

		// Wait for the Postgresql to be created.
		postgres := &postgresv1.Postgresql{}
		c.EventuallyGet(helpers.Name(name+"-database"), postgres)

		// Create a fake Postgres credentials secret.
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon." + name + "-database.credentials", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretdbpass",
			},
		}
		c.Create(dbSecret)

		// Set the status of the DB to ready.
		postgres.Status = postgresv1.ClusterStatusRunning
		c.Status().Update(postgres)

		// Set the Postgres extensions to ready.
		ext := &dbv1beta1.PostgresExtension{}
		c.EventuallyGet(helpers.Name(name+"-postgis"), ext)
		ext.Status.Status = dbv1beta1.StatusReady
		c.Status().Update(ext)
		c.EventuallyGet(helpers.Name(name+"-postgis-topology"), ext)
		ext.Status.Status = dbv1beta1.StatusReady
		c.Status().Update(ext)

		// Check that a migration Job was created.
		job := &batchv1.Job{}
		c.EventuallyGet(helpers.Name(name+"-migrations"), job)

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

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

	It("sends a single success notification on deploy", func() {
		c := helpers.TestClient

		// Advance all the various things.
		deployInstance("notifytest")

		// Check that things are ready.
		fetchInstance := &summonv1beta1.SummonPlatform{}
		c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusReady))

		// Check that the notification state saved correctly. This is mostly to wait until the final reconcile before exiting the test.
		c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyValue("80813-eb6b515-master", func(obj runtime.Object) (interface{}, error) {
			return obj.(*summonv1beta1.SummonPlatform).Status.Notification.NotifyVersion, nil
		}))

		// Find all messages since the start of the test.
		historyParams := slack.NewHistoryParameters()
		historyParams.Oldest = lastMessage.Timestamp
		history, err := slackClient.GetGroupHistory(slackChannel, historyParams)
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
		history, err := slackClient.GetGroupHistory(slackChannel, historyParams)
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
		history, err := slackClient.GetGroupHistory(slackChannel, historyParams)
		Expect(err).ToNot(HaveOccurred())
		Expect(history.Messages).To(HaveLen(2))
	})

	It("sends a single error notification on something going wrong", func() {
		c := helpers.TestClient

		// Create the SummonPlatform.
		c.Create(instance)

		// Simulate a Postgres error.
		postgres := &postgresv1.Postgresql{}
		c.EventuallyGet(helpers.Name("notifytest-database"), postgres)
		postgres.Status = postgresv1.ClusterStatusSyncFailed
		c.Status().Update(postgres)

		// Wait.
		fetchInstance := &summonv1beta1.SummonPlatform{}
		c.EventuallyGet(helpers.Name("notifytest"), fetchInstance, c.EventuallyStatus(summonv1beta1.StatusError))

		// Check that exactly one message happened
		historyParams := slack.NewHistoryParameters()
		historyParams.Oldest = lastMessage.Timestamp
		history, err := slackClient.GetGroupHistory(slackChannel, historyParams)
		Expect(err).ToNot(HaveOccurred())
		Expect(history.Messages).To(HaveLen(1))
		Expect(history.Messages[0].Attachments).To(HaveLen(1))
		Expect(history.Messages[0].Attachments[0].Color).To(Equal("a30200"))
	})
})
