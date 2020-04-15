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

package summon_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	apihelpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

const timeout = time.Second * 30

var _ = Describe("Summon controller", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		pullSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pull-secret", Namespace: helpers.OperatorNamespace}, Type: "kubernetes.io/dockerconfigjson", StringData: map[string]string{".dockerconfigjson": "{\"auths\": {}}"}}
		err := helpers.Client.Create(context.TODO(), pullSecret)
		Expect(err).NotTo(HaveOccurred())
		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"filler": []byte{}}}
		err = helpers.Client.Create(context.TODO(), appSecrets)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Display some debugging info if the test failed.
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&summonv1beta1.SummonPlatformList{})
		}
		helpers.TeardownTest()
	})

	// Minimal test, service component has no deps so it should always immediately get created.
	It("creates a service", func() {
		c := helpers.Client
		instance := &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "1.2.3",
			},
		}
		depKey := types.NamespacedName{Name: "foo-web", Namespace: helpers.Namespace}

		// Create the Summon object and expect the Reconcile and Service to be created
		err := c.Create(context.TODO(), instance)
		if apierrors.IsInvalid(err) {
			Fail(fmt.Sprintf("failed to create object, got an invalid object error: %v", err))
		}
		Expect(err).NotTo(HaveOccurred())

		service := &corev1.Service{}
		Eventually(func() error { return c.Get(context.TODO(), depKey, service) }, timeout).Should(Succeed())

		// Delete the Service and expect Reconcile to be called for Service deletion
		Expect(c.Delete(context.TODO(), service)).NotTo(HaveOccurred())
		Eventually(func() error { return c.Get(context.TODO(), depKey, service) }, timeout).Should(Succeed())
	})

	It("runs a basic reconcile", func() {
		c := helpers.TestClient
		instance := &summonv1beta1.SummonPlatform{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace}, Spec: summonv1beta1.SummonPlatformSpec{
			Version: "1.2.3",
			Secrets: []string{"testsecret"},
		}}

		// Create the SummonPlatform object and expect the Reconcile to be created.
		c.Create(instance)
		c.Status().Update(instance)

		// Check the pull_secret object.
		pullsecret := &secretsv1beta1.PullSecret{}
		c.EventuallyGet(helpers.Name("foo-pullsecret"), pullsecret)
		pullsecret.Status.Status = secretsv1beta1.StatusReady
		c.Status().Update(pullsecret)

		// Check the database object.
		db := &dbv1beta1.PostgresDatabase{}
		c.EventuallyGet(helpers.Name("foo"), db)

		// Create a fake credentials secret.
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.postgres-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretdbpass",
			},
		}
		c.Create(dbSecret)

		// Create fake aws creds from iam_user controller
		accessKey := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.aws-credentials", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("test"),
				"AWS_SECRET_ACCESS_KEY": []byte("test"),
			},
		}
		c.Create(accessKey)

		// Create fake rmq creds from rabbitmquser controller
		rmqSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretrabbitpass",
			},
		}
		c.Create(rmqSecret)

		// Set the rmq vhost to ready.
		rmqVhost := &dbv1beta1.RabbitmqVhost{}
		c.EventuallyGet(helpers.Name("foo"), rmqVhost)
		rmqVhost.Status = dbv1beta1.RabbitmqVhostStatus{
			Status: dbv1beta1.StatusReady,
			Connection: dbv1beta1.RabbitmqStatusConnection{
				Host:     "rabbitmqserver",
				Username: "foo-user",
				Vhost:    "foo",
				PasswordSecretRef: apihelpers.SecretRef{
					Name: "foo.postgres-user-password",
					Key:  "password",
				},
			},
		}
		c.Status().Update(rmqVhost)

		// Set the status of the DB to ready.
		db.Status.Status = dbv1beta1.StatusReady
		db.Status.Connection = dbv1beta1.PostgresConnection{
			Host:     "summon-qa-database",
			Username: "foo",
			Database: "foo",
			PasswordSecretRef: apihelpers.SecretRef{
				Name: "foo.postgres-user-password",
				Key:  "password",
			},
		}
		c.Status().Update(db)

		// Check that a migration Job was created.
		job := &batchv1.Job{}
		c.EventuallyGet(helpers.Name("foo-migrations"), job)

		// Mark the migrations as successful.
		job.Status.Succeeded = 1
		c.Status().Update(job)

		// Check the web Deployment object.
		deploy := &appsv1.Deployment{}
		c.EventuallyGet(helpers.Name("foo-web"), deploy)
		Expect(deploy.Spec.Replicas).To(PointTo(BeEquivalentTo(1)))
		Expect(deploy.Spec.Template.Spec.Containers).To(HaveLen(1))
		container := deploy.Spec.Template.Spec.Containers[0]
		Expect(container.Command).To(Equal([]string{"python", "-m", "summon_platform"}))
		Expect(container.Ports[0].ContainerPort).To(BeEquivalentTo(8000))
		Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/healthz"))

		// Check the web Service object.
		service := &corev1.Service{}
		c.EventuallyGet(helpers.Name("foo-web"), service)
		Expect(service.Spec.Ports[0].Port).To(BeEquivalentTo(8000))

		// Check the web Ingress object.
		ingress := &extv1beta1.Ingress{}
		c.EventuallyGet(helpers.Name("foo-web"), ingress)
		Expect(ingress.Spec.TLS[0].SecretName).To(Equal("foo-tls"))

		// Check that no hpa was deployed for celery. (celerydAuto is false by default)
		hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name("foo-celeryd-hpa"), hpa)
		}, timeout).ShouldNot(Succeed())

		// Delete the Deployment and expect it to come back.
		c.Delete(deploy)
		c.EventuallyGet(helpers.Name("foo-web"), deploy)

		// Check that component deployments are at 0 replicas by default.
		c.EventuallyGet(helpers.Name("foo-dispatch"), deploy)
		Expect(deploy.Spec.Replicas).To(PointTo(BeEquivalentTo(0)))
		c.EventuallyGet(helpers.Name("foo-businessportal"), deploy)
		Expect(deploy.Spec.Replicas).To(PointTo(BeEquivalentTo(0)))
		c.EventuallyGet(helpers.Name("foo-tripshare"), deploy)
		Expect(deploy.Spec.Replicas).To(PointTo(BeEquivalentTo(0)))
		c.EventuallyGet(helpers.Name("foo-hwaux"), deploy)
		Expect(deploy.Spec.Replicas).To(PointTo(BeEquivalentTo(0)))

		// Turn on some components and check that they have replicas.
		c.EventuallyGet(helpers.Name("foo"), instance)
		instance.Spec.Dispatch.Version = "1234"
		instance.Spec.TripShare.Version = "5678"
		// Enable HPA for celeryd
		instance.Spec.Replicas.CelerydAuto = true
		c.Update(instance)
		Eventually(func() error {
			c.Get(helpers.Name("foo-dispatch"), deploy)
			if *deploy.Spec.Replicas != 1 {
				return fmt.Errorf("No update yet")
			}
			return nil
		}, timeout).Should(Succeed())
		Eventually(func() error {
			c.Get(helpers.Name("foo-dispatch"), deploy)
			if *deploy.Spec.Replicas != 1 {
				return fmt.Errorf("No update yet")
			}
			return nil
		}, timeout).Should(Succeed())
		// celeryd-hpa should be created
		c.EventuallyGet(helpers.Name("foo-celeryd-hpa"), hpa, c.EventuallyTimeout(timeout))
		Expect(hpa.Spec.ScaleTargetRef.Name).To(Equal("foo-celeryd"))
	})

	It("reconciles labels", func() {
		c := helpers.Client
		instance := &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "1.2.3",
				Secrets: []string{"testsecret"},
			},
			Status: summonv1beta1.SummonPlatformStatus{
				MigrateVersion: "1.2.3",
			}}

		// Create the SummonPlatform object.
		err := c.Create(context.TODO(), instance)
		Expect(err).NotTo(HaveOccurred())
		err = c.Status().Update(context.TODO(), instance)
		Expect(err).NotTo(HaveOccurred())

		// Create a fake credentials secret.
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.postgres-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretdbpass",
			},
		}
		err = c.Create(context.TODO(), dbSecret)
		Expect(err).NotTo(HaveOccurred())

		// Create fake aws creds from iam_user controller
		accessKey := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.aws-credentials", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("test"),
				"AWS_SECRET_ACCESS_KEY": []byte("test"),
			},
		}
		err = c.Create(context.TODO(), accessKey)
		Expect(err).NotTo(HaveOccurred())

		// Create fake rmq creds from rabbitmquser controller
		rmqSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretrabbitpass",
			},
		}
		err = c.Create(context.TODO(), rmqSecret)
		Expect(err).NotTo(HaveOccurred())

		// Set the status of PullSecret to ready.
		pullsecret := &secretsv1beta1.PullSecret{}
		Eventually(func() error {
			return c.Get(context.TODO(), types.NamespacedName{Name: "foo-pullsecret", Namespace: helpers.Namespace}, pullsecret)
		}, timeout).
			Should(Succeed())
		pullsecret.Status.Status = secretsv1beta1.StatusReady
		err = c.Status().Update(context.TODO(), pullsecret)
		Expect(err).NotTo(HaveOccurred())

		// Set the status of the DB to ready.
		db := &dbv1beta1.PostgresDatabase{}
		Eventually(func() error {
			return c.Get(context.TODO(), types.NamespacedName{Name: "foo", Namespace: helpers.Namespace}, db)
		}, timeout).
			Should(Succeed())
		db.Status.Status = dbv1beta1.StatusReady
		db.Status.Connection = dbv1beta1.PostgresConnection{
			Host:     "foo",
			Username: "summon",
			Database: "summon",
			PasswordSecretRef: apihelpers.SecretRef{
				Name: dbSecret.Name,
				Key:  "password",
			},
		}
		err = c.Status().Update(context.TODO(), db)
		Expect(err).NotTo(HaveOccurred())

		// Set the rmq vhost to ready.
		rmqVhost := &dbv1beta1.RabbitmqVhost{}
		helpers.TestClient.EventuallyGet(helpers.Name("foo"), rmqVhost)
		rmqVhost.Status = dbv1beta1.RabbitmqVhostStatus{
			Status: dbv1beta1.StatusReady,
			Connection: dbv1beta1.RabbitmqStatusConnection{
				Host:     "rabbitmqserver",
				Username: "foo-user",
				Vhost:    "foo",
				PasswordSecretRef: apihelpers.SecretRef{
					Name: "foo.rabbitmq-user-password",
					Key:  "password",
				},
			},
		}
		helpers.TestClient.Status().Update(rmqVhost)

		// Fetch the Deployment and check the initial label.
		deploy := &appsv1.Deployment{}
		Eventually(func() error {
			return c.Get(context.TODO(), types.NamespacedName{Name: "foo-web", Namespace: helpers.Namespace}, deploy)
		}, timeout).
			Should(Succeed())
		Expect(deploy.Labels["app.kubernetes.io/instance"]).To(Equal("foo-web"))
		Expect(deploy.Labels).ToNot(HaveKey("other"))

		// Modify the labels.
		deploy.Labels["app.kubernetes.io/instance"] = "boom"
		deploy.Labels["other"] = "foo"
		err = c.Update(context.TODO(), deploy)
		Expect(err).NotTo(HaveOccurred())

		// Check that the labels end up correct.
		Eventually(func() error {
			err := c.Get(context.TODO(), types.NamespacedName{Name: "foo-web", Namespace: helpers.Namespace}, deploy)
			if err != nil {
				return err
			}
			value, ok := deploy.Labels["other"]
			if !ok || value != "foo" {
				return fmt.Errorf("No update yet (other)")
			}
			return nil
		}, timeout).Should(Succeed())
		Eventually(func() error {
			err := c.Get(context.TODO(), types.NamespacedName{Name: "foo-web", Namespace: helpers.Namespace}, deploy)
			if err != nil {
				return err
			}
			value, ok := deploy.Labels["app.kubernetes.io/instance"]
			if !ok || value != "foo-web" {
				return fmt.Errorf("No update yet (app.kubernetes.io/instance)")
			}
			return nil
		}, timeout).Should(Succeed())
	})

	It("manages the status correctly", func() {
		c := helpers.Client

		// Create a SummonPlatform and related objects.
		instance := &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "statustester", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "1-abcdef1-master",
				Secrets: []string{"statustester"},
			},
		}
		err := c.Create(context.TODO(), instance)
		Expect(err).NotTo(HaveOccurred())
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "statustester.postgres-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretdbpass",
			},
		}
		err = c.Create(context.TODO(), dbSecret)
		Expect(err).NotTo(HaveOccurred())
		// Create fake aws creds from iam_user controller
		accessKey := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "statustester.aws-credentials", Namespace: helpers.Namespace},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("test"),
				"AWS_SECRET_ACCESS_KEY": []byte("test"),
			},
		}
		err = c.Create(context.TODO(), accessKey)
		Expect(err).NotTo(HaveOccurred())

		// Create fake rmq creds from rabbitmquser controller
		rmqSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "statustester.rabbitmq-user-password", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretrabbitpass",
			},
		}
		err = c.Create(context.TODO(), rmqSecret)

		Expect(err).NotTo(HaveOccurred())
		inSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "statustester", Namespace: helpers.Namespace},
			StringData: map[string]string{},
		}
		err = c.Create(context.TODO(), inSecret)
		Expect(err).NotTo(HaveOccurred())

		// Wait for the database to be created.
		db := &dbv1beta1.PostgresDatabase{}
		Eventually(func() error {
			return c.Get(context.TODO(), types.NamespacedName{Name: "statustester", Namespace: helpers.Namespace}, db)
		}, timeout).Should(Succeed())

		// Check the status. Should not be set yet.
		assertStatus := func(status string) {
			Eventually(func() (string, error) {
				err := c.Get(context.TODO(), types.NamespacedName{Name: "statustester", Namespace: helpers.Namespace}, instance)
				return instance.Status.Status, err
			}, timeout).Should(Equal(status))
		}
		assertStatus("")

		// Set the database to Running
		db.Status.Status = dbv1beta1.StatusReady
		db.Status.Connection = dbv1beta1.PostgresConnection{
			Host:     "summon-dev-database",
			Username: "statustester_dev",
			Database: "statustester_dev",
			PasswordSecretRef: apihelpers.SecretRef{
				Name: "statustester.postgres-user-password",
				Key:  "password",
			},
		}
		err = c.Status().Update(context.TODO(), db)
		Expect(err).NotTo(HaveOccurred())

		// Check the status. Should still be Initializing.
		assertStatus(summonv1beta1.StatusInitializing)

		// Set the pull secret to ready.
		pullSecret := &secretsv1beta1.PullSecret{}
		Eventually(func() error {
			return c.Get(context.TODO(), types.NamespacedName{Name: "statustester-pullsecret", Namespace: helpers.Namespace}, pullSecret)
		}, timeout).Should(Succeed())
		pullSecret.Status.Status = secretsv1beta1.StatusReady
		err = c.Status().Update(context.TODO(), pullSecret)
		Expect(err).NotTo(HaveOccurred())

		// Set the rmq vhost to ready.
		rmqVhost := &dbv1beta1.RabbitmqVhost{}
		helpers.TestClient.EventuallyGet(helpers.Name("statustester"), rmqVhost)
		rmqVhost.Status = dbv1beta1.RabbitmqVhostStatus{
			Status: dbv1beta1.StatusReady,
			Connection: dbv1beta1.RabbitmqStatusConnection{
				Host:     "rabbitmqserver",
				Username: "statustester-user",
				Vhost:    "statustester",
				PasswordSecretRef: apihelpers.SecretRef{
					Name: "statustester.rabbitmq-user-password",
					Key:  "password",
				},
			},
		}
		helpers.TestClient.Status().Update(rmqVhost)

		// Check the status again. Should be Migrating.
		assertStatus(summonv1beta1.StatusMigrating)

		// Mark the migration as a success.
		job := &batchv1.Job{}
		Eventually(func() error {
			return c.Get(context.TODO(), types.NamespacedName{Name: "statustester-migrations", Namespace: helpers.Namespace}, job)
		}, timeout).Should(Succeed())
		job.Status.Succeeded = 1
		err = c.Status().Update(context.TODO(), job)
		Expect(err).NotTo(HaveOccurred())

		// Check the status again. Should be Deploying.
		assertStatus(summonv1beta1.StatusDeploying)

		// Set deployments and statefulsets to ready.
		updateDeployment := func(s string) {
			deployment := &appsv1.Deployment{}
			Eventually(func() error {
				return c.Get(context.TODO(), types.NamespacedName{Name: "statustester-" + s, Namespace: helpers.Namespace}, deployment)
			}, timeout).Should(Succeed())
			deployment.Status.Replicas = 1
			deployment.Status.ReadyReplicas = 1
			deployment.Status.AvailableReplicas = 1
			err = c.Status().Update(context.TODO(), deployment)
			Expect(err).NotTo(HaveOccurred())
		}
		updateStatefulSet := func(s string) {
			statefulset := &appsv1.StatefulSet{}
			Eventually(func() error {
				return c.Get(context.TODO(), types.NamespacedName{Name: "statustester-" + s, Namespace: helpers.Namespace}, statefulset)
			}, timeout).Should(Succeed())
			Expect(err).NotTo(HaveOccurred())
			statefulset.Status.Replicas = 1
			statefulset.Status.ReadyReplicas = 1
			err = c.Status().Update(context.TODO(), statefulset)
			Expect(err).NotTo(HaveOccurred())
		}
		updateDeployment("web")
		updateDeployment("daphne")
		updateDeployment("celeryd")
		updateDeployment("channelworker")
		updateDeployment("static")
		updateStatefulSet("celerybeat")

		// Check the status again. Should be Deploying.
		assertStatus(summonv1beta1.StatusReady)
	})

	It("block all action with skip-reconcile annotation", func() {
		c := helpers.Client
		instance := &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "annotest",
				Namespace: helpers.Namespace,
				Annotations: map[string]string{
					"ridecell.io/skip-reconcile": "true",
				},
			},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "1.2.3",
			},
		}

		// Create the SummonPlatform object and expect the Reconcile to be created.
		err := c.Create(context.TODO(), instance)
		Expect(err).ToNot(HaveOccurred())

		// Check whether a service obj is created
		service := &corev1.Service{}

		Consistently(func() error {
			return c.Get(context.TODO(), helpers.Name("annotest-web"), service)
		}, time.Second*10).ShouldNot(Succeed())
	})
})
