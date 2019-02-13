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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("Summon controller appsecrets", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *summonv1beta1.SummonPlatform

	// Test helper functions.
	getData := func(obj runtime.Object) (interface{}, error) {
		secret := obj.(*corev1.Secret)
		data := map[string]interface{}{}
		err := yaml.Unmarshal(secret.Data["summon-platform.yml"], &data)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	createInputSecret := func() *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"TOKEN": "secrettoken",
			},
		}
		helpers.TestClient.Create(secret)
		return secret
	}

	createAwsSecret := func() *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "appsecretstest.aws-credentials", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIAtest",
				"AWS_SECRET_ACCESS_KEY": "test",
			},
		}
		helpers.TestClient.Create(secret)
		return secret
	}

	createDbSecret := func() *corev1.Secret {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "summon.appsecretstest-database.credentials", Namespace: helpers.Namespace},
			StringData: map[string]string{
				"password": "secretdbpass",
			},
		}
		helpers.TestClient.Create(secret)
		return secret
	}

	createInstance := func() {
		instance.Spec.Secrets = []string{"testsecret"}
		helpers.TestClient.Create(instance)

		// Advance postgres to running.
		postgres := &postgresv1.Postgresql{}
		helpers.TestClient.EventuallyGet(helpers.Name("appsecretstest-database"), postgres)
		postgres.Status = postgresv1.ClusterStatusRunning
		helpers.TestClient.Status().Update(postgres)
	}

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()

		// Set up the instance object for other tests.
		instance = &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{Name: "appsecretstest", Namespace: helpers.Namespace},
			Spec: summonv1beta1.SummonPlatformSpec{
				Version: "80813-eb6b515-master",
				Secrets: []string{},
				Database: summonv1beta1.DatabaseSpec{
					ExclusiveDatabase: true,
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

	It("creates the app secret if all inputs exist already", func() {
		c := helpers.TestClient

		// Create all the input secrets.
		createInputSecret()
		createAwsSecret()
		createDbSecret()

		// Create the instance.
		createInstance()

		// Get the output app secrets.
		appSecret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("appsecretstest.app-secrets"), appSecret)

		// Parse the YAML to check it.
		data := map[string]interface{}{}
		err := yaml.Unmarshal(appSecret.Data["summon-platform.yml"], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data["DATABASE_URL"]).To(Equal("postgis://summon:secretdbpass@appsecretstest-database/summon"))
		Expect(data["TOKEN"]).To(Equal("secrettoken"))
	})

	It("creates the app secret if the database secret is created afterwards", func() {
		c := helpers.TestClient

		// Create some of the input secrets.
		createInputSecret()
		createAwsSecret()

		// Create the instance.
		createInstance()

		// Create the DB secret later than where it would normally be created.
		time.Sleep(2 * time.Second)
		createDbSecret()

		// Get the output app secrets.
		appSecret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("appsecretstest.app-secrets"), appSecret)

		// Parse the YAML to check it.
		data := map[string]interface{}{}
		err := yaml.Unmarshal(appSecret.Data["summon-platform.yml"], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data["DATABASE_URL"]).To(Equal("postgis://summon:secretdbpass@appsecretstest-database/summon"))
	})

	It("updates the app secret if the database secret is changed afterwards", func() {
		c := helpers.TestClient

		// Create the input secrets.
		createInputSecret()
		dbSecret := createDbSecret()
		createAwsSecret()

		// Create the instance.
		createInstance()

		// Change the DB secret
		time.Sleep(10 * time.Second)
		dbSecret.StringData["password"] = "other"
		c.Update(dbSecret)

		// Get the output app secrets.
		appSecret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("appsecretstest.app-secrets"), appSecret, c.EventuallyValue(HaveKeyWithValue("DATABASE_URL", "postgis://summon:other@appsecretstest-database/summon"), getData))
	})

	It("errors if the input secret does not exist", func() {
		c := helpers.TestClient

		// Create some of the input secrets.
		createDbSecret()
		createAwsSecret()

		// Create the instance.
		createInstance()

		// Check the status.
		c.EventuallyGet(helpers.Name("appsecretstest"), instance, c.EventuallyStatus(summonv1beta1.StatusError))
	})

	It("updates the app secrets if an input secret is changed afterwards", func() {
		c := helpers.TestClient

		// Create the input secrets.
		inputSecret := createInputSecret()
		createDbSecret()
		createAwsSecret()

		// Create the instance.
		createInstance()

		// Change the DB secret
		time.Sleep(10 * time.Second)
		inputSecret.StringData["TOKEN"] = "other"
		c.Update(inputSecret)

		// Get the output app secrets.
		appSecret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("appsecretstest.app-secrets"), appSecret, c.EventuallyValue(HaveKeyWithValue("TOKEN", "other"), getData))
	})
})
