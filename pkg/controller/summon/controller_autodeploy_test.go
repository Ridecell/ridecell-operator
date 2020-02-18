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
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/heroku/docker-registry-client/registry"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	gcr "github.com/Ridecell/ridecell-operator/pkg/utils/gcr"
)

func addMockTags(tags []string) error {
	registry_url := os.Getenv("LOCAL_REGISTRY_URL")
	if registry_url == "" {
		return errors.New("No mock registry found to upload tags!")
	}

	// Setup hub connection
	var key = os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")
	var transport = registry.WrapTransport(http.DefaultTransport, registry_url, "_json_key", key)
	var summonHub = &registry.Registry{
		URL: registry_url,
		Client: &http.Client{
			Transport: transport,
		},
		Logf: registry.Quiet,
	}

	// Get the base manifest in our mock registry to create mock tags.
	manifest, err := summonHub.ManifestV2("ridecell-1/summon", "basetag")

	if err != nil {
		return err
	}

	for _, tag := range tags {
		err := summonHub.PutManifest("ridecell-1/summon", tag, manifest)
		if err != nil {
			fmt.Printf("ERROR adding %s to registry: %s", tag, err)
		}
	}
	return nil
}

var _ = Describe("Summon controller autodeploy @autodeploy", func() {
	var instance *summonv1beta1.SummonPlatform
	var helpers *test_helpers.PerTestHelpers

	// Start the registry with some default image tags
	MockTags := []string{"1-abc1234-test-branch", "2-def5678-test-branch", "1-abc1234-other-branch"}
	_ = addMockTags(MockTags)

	BeforeEach(func() {
		registry_url := os.Getenv("LOCAL_REGISTRY_URL")
		if registry_url == "" {
			Skip("Skipping Autodeploy controller tests -- no local docker registry to test against.")
		}

		helpers = testHelpers.SetupTest()

		// Setup secrets summonplatform
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

		instance = &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Secrets: []string{"testsecret"},
			},
		}
		// reset the cache update timer
		gcr.LastCacheUpdate = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			summons := &summonv1beta1.SummonPlatformList{}
			err := helpers.Client.List(context.Background(), nil, summons)
			if err != nil {
				fmt.Printf("!!!!!! %s\n", err)
			} else {
				fmt.Print("Failed instances:\n")
				for _, item := range summons.Items {
					if item.Namespace == helpers.Namespace {
						fmt.Printf("\t%s %#v\n", item.Name, item)
					}
				}
			}
		}
		helpers.TeardownTest()
	})

	setupDeployPrereqs := func(name string) {
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
	}

	Context("gcr utility component", func() {
		It("periodically updates the tag cache (per CacheExpiry)", func() {
			instance.Spec.AutoDeploy = "test-branch"
			setupDeployPrereqs("foo")

			// basetag comes from initial registry setup
			tagState := append(MockTags, "basetag")
			// circleci runs tests in random order, and registry may pick up tags from other
			// test cases, so at least confirm these tags exist.
			for _, tag := range tagState {
				Expect(gcr.CachedTags).To(ContainElement(tag))
			}

			newtags := []string{"gcr-update-test"}
			_ = addMockTags(newtags)

			// set LastCacheUpdate time to 5 mins in the past instead of waiting to mock cacheExpiry period
			// and confirm cache update occurs.
			gcr.LastCacheUpdate = time.Now().Add(time.Minute * -5)

			// Still need to give gcr utility a little time to update Cachetag
			time.Sleep(time.Second * 5)
			Expect(gcr.CachedTags).To(ContainElement("gcr-update-test"))
		})
	})

	It("deploys latest image of branch specified in autodeploy", func() {
		c := helpers.TestClient
		instance.Spec.AutoDeploy = "TestTag"
		tags := []string{"11-95ac60f-TestTag", "15-ab0f6c1-TestTag", "16-de0a8fb-TestTag2"}
		_ = addMockTags(tags)
		setupDeployPrereqs("foo")

		// Check that a migration Job was created.
		job := &batchv1.Job{}
		c.EventuallyGet(helpers.Name("foo-migrations"), job)

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

		//Expect deployment to deploy with latest branch tag
		deployment := &appsv1.Deployment{}
		c.EventuallyGet(helpers.Name("foo-web"), deployment)
		Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("us.gcr.io/ridecell-1/summon:15-ab0f6c1-TestTag"))
		// Autodeploy doesn't modify the actual summonplatform spec
		Expect(instance.Spec.Version).To(Equal(""))
	})

	It("triggers autodeploy reconcile when channel gets event (via summonplatform-controller watcher)", func() {
		c := helpers.TestClient
		instance.Spec.AutoDeploy = "devops-feature-test"
		tags := []string{"154551-2634073-devops-feature-test", "154480-bc4c502-devops-feature-test"}
		_ = addMockTags(tags)
		setupDeployPrereqs("foo")

		// Check that a migration Job was created.
		job := &batchv1.Job{}
		c.EventuallyGet(helpers.Name("foo-migrations"), job)

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

		// Expect the deployment to be created with the latest branch tag.
		deployment := &appsv1.Deployment{}
		c.EventuallyGet(helpers.Name("foo-web"), deployment)
		Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("us.gcr.io/ridecell-1/summon:154551-2634073-devops-feature-test"))

		// Simulate new docker image upload and allow wait time < 5min before cache refresh.
		_ = addMockTags([]string{"154575-cdf9c69-devops-feature-test"})
		waitTimer := time.Now().Add(time.Second * 15)
		for time.Since(waitTimer) < 0 {
		}

		// There should have been no updates to main tag cache yet.
		Expect(gcr.LastCacheUpdate).Should(BeTemporally("<", time.Now()))

		// Re-fetch the deployment object and check that there was no change to Spec.Version used.
		c.EventuallyGet(helpers.Name("foo-web"), deployment)
		Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("us.gcr.io/ridecell-1/summon:154551-2634073-devops-feature-test"))

		// Confirm cache tag gets updated. (Results from controller sending event and triggering autodeploy reconcile)
		c.EventuallyGet(helpers.Name("foo-migrations"), job, c.EventuallyValue(
			Equal("us.gcr.io/ridecell-1/summon:154575-cdf9c69-devops-feature-test"),
			func(obj runtime.Object) (interface{}, error) {
				return obj.(*batchv1.Job).Spec.Template.Spec.Containers[0].Image, nil
			}), c.EventuallyTimeout(time.Minute))

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

		// Check autodeploy reconcile resulted in deploying to latest image of branch.
		c.EventuallyGet(helpers.Name("foo-web"), deployment, c.EventuallyValue(Equal("us.gcr.io/ridecell-1/summon:154575-cdf9c69-devops-feature-test"), func(obj runtime.Object) (interface{}, error) {
			return obj.(*appsv1.Deployment).Spec.Template.Spec.Containers[0].Image, nil
		}))
	})

	It("sets error status and message if Spec.Version and Spec.AutoDeploy not specified", func() {
		c := helpers.TestClient
		c.Create(instance)

		summonplatform := &summonv1beta1.SummonPlatform{}
		c.EventuallyGet(helpers.Name("foo"), summonplatform, c.EventuallyStatus(summonv1beta1.StatusError))
		Expect(summonplatform.Status.Status).To(Equal(summonv1beta1.StatusError))
		Expect(summonplatform.Status.Message).To(Equal("Spec.Version OR Spec.AutoDeploy must be set. No Version set for deployment."))

	})

	It("sets error status and message if Spec.Version and Spec.AutoDeploy both specified", func() {
		c := helpers.TestClient
		instance.Spec.Version = "1.2.3"
		instance.Spec.AutoDeploy = "test-branch"
		c.Create(instance)

		summonplatform := &summonv1beta1.SummonPlatform{}
		c.EventuallyGet(helpers.Name("foo"), summonplatform, c.EventuallyStatus(summonv1beta1.StatusError))
		Expect(summonplatform.Status.Status).To(Equal(summonv1beta1.StatusError))
		Expect(summonplatform.Status.Message).To(Equal("Spec.Version and Spec.AutoDeploy are both set. Must specify only one."))
	})

	It("errors if no docker image found for branch", func() {
		c := helpers.TestClient
		// Create the SummonPlatform.
		instance.Name = "foo"
		instance.Spec.AutoDeploy = "devops-non-existent-branch"
		instance.ResourceVersion = ""
		c.Create(instance)

		// PullSecret won't be created because AutoDeploy runs into error.
		pullsecret := &secretsv1beta1.PullSecret{}

		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name("foo-pullsecret"), pullsecret)
		}, timeout).ShouldNot(Succeed())

		Expect(instance.Spec.Version).To(Equal(""))

		//update instance to get latest status and check
		c.Status().Update(instance)
		c.EventuallyGet(helpers.Name("foo"), instance, c.EventuallyStatus(summonv1beta1.StatusError))
		Eventually(instance.Status.Message).Should(Equal("autodeploy: no matching branch image for devops-non-existent-branch"))
	})
})
