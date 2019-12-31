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

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	//"github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	"github.com/heroku/docker-registry-client/registry"
	//"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	gcr "github.com/Ridecell/ridecell-operator/pkg/utils/gcr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

	repositories, err := summonHub.Repositories()
	fmt.Printf("DEBUG: DOCKER: Repositories seen: %+v\n", repositories)
	// Get the base manifest in our mock registry to create mock tags.
	manifest, err := summonHub.ManifestV2("ridecell-1/summon", "basetag")

	if err != nil {
		return err
	}

	for _, tag := range tags {
		summonHub.PutManifest("ridecell-1/summon", tag, manifest)
	}

	return nil
}

var _ = FDescribe("Summon controller autodeploy @autodeploy", func() {
	var instance *summonv1beta1.SummonPlatform
	var helpers *test_helpers.PerTestHelpers

	// Start the registry with some default image tags
	MockTags := []string{"1-abc1234-test-branch", "2-def5678-test-branch", "1-abc1234-other-branch"}
	err := addMockTags(MockTags)
	if err != nil {
		fmt.Printf("ERROR: Unable to addMockTags %s\n", err)
	}

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
		// clean up test case tags from registry? Can't unless registry configured to allow this.
		/*
			digest, err := hub.ManifestDigest("heroku/cedar", "14")
			err = hub.DeleteManifest("heroku/cedar", digest)
		*/
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

	/* It("try to delete image", func() {
		var key = os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")
		var transport = registry.WrapTransport(http.DefaultTransport, "http://localhost:500", "_json_key", key)
		var summonHub = &registry.Registry{
			URL: "http://localhost:5000",
			Client: &http.Client{
				Transport: transport,
			},
			Logf: registry.Quiet,
		}
		digest, err := summonHub.ManifestDigest("ridecell-1/summon", "154551-2634073-devops-feature-test")
		err = summonHub.DeleteManifest("ridecell-1/summon", digest)
		fmt.Printf("Any error deleting tag? %v\n", err)
		tags, err := summonHub.Tags("ridecell-1/summon")
		fmt.Printf("DEBUG: Now tag list is %+v\n", tags)
	})
	*/

	It("deploys latest image of branch specified in autodeploy", func() {
		c := helpers.TestClient
		instance.Spec.AutoDeploy = "TestTag"
		tags := []string{"11-95ac60f-TestTag", "15-ab0f6c1-TestTag", "16-de0a8fb-TestTag2"}
		addMockTags(tags)
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

	It("triggers autodeploy reconcile when tag cache updated (via watcher)", func() {
		c := helpers.TestClient
		instance.Spec.AutoDeploy = "devops-feature-test"
		tags := []string{"154551-2634073-devops-feature-test", "154480-bc4c502-devops-feature-test"}
		addMockTags(tags)
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

		// Simulate new docker image upload and allow sleep time < 5min before cache refresh.
		addMockTags([]string{"154575-cdf9c69-devops-feature-test"})
		time.Sleep(time.Second * 15)

		// There should have been no updates to main tag cache.
		Expect(gcr.LastCacheUpdate).Should(BeTemporally("<", time.Now()))
		// Re-fetch the deployment object and check that there was no change to Spec.Version used.
		c.EventuallyGet(helpers.Name("foo-web"), deployment)
		Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("us.gcr.io/ridecell-1/summon:154551-2634073-devops-feature-test"))

		// Instead of actually waiting 5 minutes for gcr.CachedTags to get updated which triggers
		// the watcher to queue up autodeploy reconcile, simulate an updated CacheTag by directly modifying it
		// and update LastCacheUpdate to now time.
		gcr.CachedTags = append(gcr.CachedTags, "154575-cdf9c69-devops-feature-test")
		gcr.LastCacheUpdate = time.Now()

		// Need to give operators a few second to delete old job before fetching newly
		// created one.
		waitTimer := time.Now().Add(time.Second * 3)

		// Wait instead of sleep since we need goroutines to keep running
		for time.Since(waitTimer) < 0 {
		}

		// Check that another migration Job was created and its the new version.
		c.EventuallyGet(helpers.Name("foo-migrations"), job)
		Eventually(func() string {
			return job.Spec.Template.Spec.Containers[0].Image
		}, time.Minute).Should(Equal("us.gcr.io/ridecell-1/summon:154575-cdf9c69-devops-feature-test"))

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

		// Also need to wait a bit for deployment to get updated.
		waitTimer = time.Now().Add(time.Second * 3)
		for time.Since(waitTimer) < 0 {
		}

		// Expect another deployment to deploy with latest branch tag
		c.EventuallyGet(helpers.Name("foo-web"), deployment)
		Eventually(func() string {
			return deployment.Spec.Template.Spec.Containers[0].Image
		}, timeout).Should(Equal("us.gcr.io/ridecell-1/summon:154575-cdf9c69-devops-feature-test"))
	})

	It("does not deploy if neither spec version or autodeploy is specified", func() {
		c := helpers.TestClient
		setupDeployPrereqs("foo")

		// Don't expect migrations to occur.
		job := &batchv1.Job{}
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name("foo-migrations"), job)
		}, timeout).ShouldNot(Succeed())

		Expect(instance.Spec.Version).To(Equal(""))
		Expect(instance.Spec.AutoDeploy).To(Equal(""))

		//update instance to get latest status and check
		c.Status().Update(instance)
		c.EventuallyGet(helpers.Name("foo"), instance, c.EventuallyStatus(summonv1beta1.StatusError))
		Eventually(instance.Status.Message).Should(Equal("Spec.Version OR Spec.AutoDeploy must be set. No Version set for deployment."))
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
