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

package components_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("app_secrets Component", func() {
	var inSecret, postgresSecret, fernetKeys, rabbitmqPassword, secretKey, accessKey *corev1.Secret
	var comp components.Component

	BeforeEach(func() {
		instance.Status.PostgresStatus = dbv1beta1.StatusReady
		instance.Status.PostgresConnection = dbv1beta1.PostgresConnection{
			Host:     "summon-qa-database",
			Username: "foo_qa",
			Database: "foo_qa",
			PasswordSecretRef: helpers.SecretRef{
				Name: "foo-qa.postgres-user-password",
				Key:  "password",
			},
		}
		instance.Status.RabbitMQStatus = dbv1beta1.StatusReady
		instance.Status.RabbitMQConnection = dbv1beta1.RabbitmqStatusConnection{
			Host:     "rabbitmqserver",
			Username: "foo-user",
			Vhost:    "foo",
			PasswordSecretRef: helpers.SecretRef{
				Name: "foo.rabbitmq-user-password",
				Key:  "password",
			},
		}

		inSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret", Namespace: "default"},
			Data: map[string][]byte{
				"TOKEN": []byte("secrettoken"),
			},
		}

		formattedTime := time.Time.Format(time.Now().UTC(), summoncomponents.CustomTimeLayout)
		fernetKeys = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.fernet-keys", Namespace: "default"},
			Data: map[string][]byte{
				formattedTime: []byte("lorem ipsum"),
			},
		}

		postgresSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-qa.postgres-user-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("postgresPassword"),
			},
		}

		secretKey = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.secret-key", Namespace: "default"},
			Data: map[string][]byte{
				"SECRET_KEY": []byte("testkey"),
			},
		}

		accessKey = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.aws-credentials", Namespace: "default"},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("testid"),
				"AWS_SECRET_ACCESS_KEY": []byte("testkey"),
			},
		}

		rabbitmqPassword = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("rabbitmqpassword"),
			},
		}

		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		comp = summoncomponents.NewAppSecret()
	})

	It("Unreconcilable when db not ready", func() {
		instance.Status.PostgresStatus = ""
		Expect(comp.IsReconcilable(ctx)).To(Equal(false))
	})

	It("Reconcilable when db is ready", func() {
		Expect(comp.IsReconcilable(ctx)).To(Equal(true))
	})

	It("Run reconcile without a postgres password", func() {
		ctx.Client = fake.NewFakeClient(inSecret, fernetKeys, secretKey, accessKey)
		Expect(comp).ToNot(ReconcileContext(ctx))
	})

	It("Run reconcile with a blank postgres password", func() {
		delete(postgresSecret.Data, "password")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		_, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError("app_secrets: Postgres password not found in secret"))
	})

	It("Sets postgres password and checks reconcile output", func() {
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "foo.app-secrets", Namespace: "default"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		var parsedYaml map[string]interface{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &parsedYaml)
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedYaml["DATABASE_URL"]).To(Equal("postgis://foo_qa:postgresPassword@summon-qa-database/foo_qa"))
		Expect(parsedYaml["OUTBOUNDSMS_URL"]).To(Equal("https://foo.prod.ridecell.io/outbound-sms"))
		Expect(parsedYaml["SMS_WEBHOOK_URL"]).To(Equal("https://foo.ridecell.us/sms/receive/"))
		Expect(parsedYaml["CELERY_BROKER_URL"]).To(Equal("pyamqp://foo-user:rabbitmqpassword@rabbitmqserver/foo?ssl=true"))
		Expect(parsedYaml["TOKEN"]).To(Equal("secrettoken"))
		Expect(parsedYaml["AWS_ACCESS_KEY_ID"]).To(Equal("testid"))
		Expect(parsedYaml["AWS_SECRET_ACCESS_KEY"]).To(Equal("testkey"))
	})

	It("copies data from the input secret", func() {
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "foo.app-secrets", Namespace: "default"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		var parsedYaml map[string]interface{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &parsedYaml)
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedYaml["TOKEN"]).To(Equal("secrettoken"))
	})

	It("reconciles with existing fernet keys", func() {
		now := time.Now().UTC()
		addKey := func(k, v string) {
			d, _ := time.ParseDuration(k)
			fernetKeys.Data[time.Time.Format(now.Add(d), summoncomponents.CustomTimeLayout)] = []byte(v)
		}
		fernetKeys.Data = map[string][]byte{}
		addKey("-1h", "1")
		addKey("-2h", "2")
		addKey("-3h", "3")
		addKey("-4h", "4")
		addKey("-5h", "5")
		addKey("-6h", "6")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)

		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "foo.app-secrets", Namespace: "default"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		var parsedYaml struct {
			Keys []string `yaml:"FERNET_KEYS"`
		}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &parsedYaml)
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedYaml.Keys).To(Equal([]string{"1", "2", "3", "4", "5", "6"}))
	})

	It("runs reconcile with no secret_key", func() {
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys)
		res, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError(`app_secrets: error fetching derived app secret foo.secret-key: secrets "foo.secret-key" not found`))
		Expect(res.Requeue).To(BeTrue())
	})

	It("runs reconcile with all values set", func() {
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("overwrites values using multiple secrets", func() {
		instance.Spec.Secrets = []string{"testsecret", "testsecret2", "testsecret3"}

		inSecret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret2", Namespace: "default"},
			Data: map[string][]byte{
				"TOKEN": []byte("overwritten"),
			},
		}

		inSecret3 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret3", Namespace: "default"},
			Data: map[string][]byte{
				"TOKEN": []byte("overwritten_again"),
			},
		}

		ctx.Client = fake.NewFakeClient(inSecret, inSecret2, inSecret3, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo.app-secrets", Namespace: "default"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		appSecretsData := map[string]interface{}{}

		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &appSecretsData)
		Expect(err).ToNot(HaveOccurred())
		Expect(appSecretsData["TOKEN"]).To(Equal("overwritten_again"))

	})
})
