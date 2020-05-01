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
	"encoding/json"
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
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
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
			Host:     "summon-dev-database",
			Username: "foo_dev",
			Database: "foo_dev",
			PasswordSecretRef: helpers.SecretRef{
				Name: "foo-dev.postgres-user-password",
				Key:  "password",
			},
		}
		instance.Status.RabbitMQStatus = dbv1beta1.StatusReady
		instance.Status.RabbitMQConnection = dbv1beta1.RabbitmqStatusConnection{
			Host:     "rabbitmqserver",
			Username: "foo-dev-user",
			Vhost:    "foo-dev",
			PasswordSecretRef: helpers.SecretRef{
				Name: "foo-dev.rabbitmq-user-password",
				Key:  "password",
			},
		}

		inSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"TOKEN": []byte("secrettoken"),
			},
		}

		formattedTime := time.Time.Format(time.Now().UTC(), summoncomponents.CustomTimeLayout)
		fernetKeys = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev.fernet-keys", Namespace: "summon-dev"},
			Data: map[string][]byte{
				formattedTime: []byte("lorem ipsum"),
			},
		}

		postgresSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev.postgres-user-password", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"password": []byte("postgresPassword"),
			},
		}

		secretKey = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev.secret-key", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"SECRET_KEY": []byte("testkey"),
			},
		}

		accessKey = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev.aws-credentials", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("testid"),
				"AWS_SECRET_ACCESS_KEY": []byte("testkey"),
			},
		}

		rabbitmqPassword = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev.rabbitmq-user-password", Namespace: "summon-dev"},
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

	It("Run reconcile with a missing postgres password", func() {
		delete(postgresSecret.Data, "password")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		_, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError("app_secrets: Postgres secret not initialized"))
	})

	It("Run reconcile with a missing postgres password and other fields", func() {
		delete(postgresSecret.Data, "password")
		postgresSecret.Data["foo"] = []byte("other")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		_, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError("app_secrets: Postgres password not found in secret"))
	})

	It("Sets postgres password and checks reconcile output", func() {
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		var parsedYaml map[string]interface{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &parsedYaml)
		Expect(err).ToNot(HaveOccurred())

		Expect(parsedYaml["DATABASE_URL"]).To(Equal("postgis://foo_dev:postgresPassword@summon-dev-database/foo_dev"))
		Expect(parsedYaml["OUTBOUNDSMS_URL"]).To(Equal("https://foo-dev.prod.ridecell.io/outbound-sms"))
		Expect(parsedYaml["SMS_WEBHOOK_URL"]).To(Equal("https://foo-dev.ridecell.us/sms/receive/"))
		Expect(parsedYaml["CELERY_BROKER_URL"]).To(Equal("pyamqp://foo-dev-user:rabbitmqpassword@rabbitmqserver/foo-dev?ssl=true"))
		Expect(parsedYaml["TOKEN"]).To(Equal("secrettoken"))
		Expect(parsedYaml["AWS_ACCESS_KEY_ID"]).To(Equal("testid"))
		Expect(parsedYaml["AWS_SECRET_ACCESS_KEY"]).To(Equal("testkey"))
	})

	It("copies data from the input secret", func() {
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
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
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
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
		Expect(err).To(MatchError(`app_secrets: error fetching derived app secret foo-dev.secret-key: secrets "foo-dev.secret-key" not found`))
		Expect(res.Requeue).To(BeTrue())
	})

	It("runs reconcile with all values set", func() {
		Expect(comp).To(ReconcileContext(ctx))
	})

	It("overwrites values using multiple secrets", func() {
		instance.Spec.Secrets = []string{"testsecret", "testsecret2", "testsecret3"}

		inSecret2 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret2", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"TOKEN": []byte("overwritten"),
			},
		}

		inSecret3 := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "testsecret3", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"TOKEN": []byte("overwritten_again"),
			},
		}

		ctx.Client = fake.NewFakeClient(inSecret, inSecret2, inSecret3, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		appSecretsData := map[string]interface{}{}

		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &appSecretsData)
		Expect(err).ToNot(HaveOccurred())
		Expect(appSecretsData["TOKEN"]).To(Equal("overwritten_again"))
	})

	It("handles a SAML_PRIVATE_KEY", func() {
		inSecret.Data["SAML_PRIVATE_KEY"] = []byte("supersecretprivatekey")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		appSecretsData := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &appSecretsData)
		Expect(err).ToNot(HaveOccurred())
		Expect(appSecretsData).To(HaveKeyWithValue("SAML_PRIVATE_KEY_FILENAME", "sp.key"))
		Expect(appSecretsData).ToNot(HaveKey("SAML_PUBLIC_KEY_FILENAME"))

		err = ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.saml", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchSecret.Data).To(HaveKeyWithValue("sp.key", []byte("supersecretprivatekey")))
	})

	It("handles a SAML_PUBLIC_KEY", func() {
		inSecret.Data["SAML_PUBLIC_KEY"] = []byte("veryverifiedcert")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		appSecretsData := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &appSecretsData)
		Expect(err).ToNot(HaveOccurred())
		Expect(appSecretsData).To(HaveKeyWithValue("SAML_PUBLIC_KEY_FILENAME", "sp.crt"))
		Expect(appSecretsData).ToNot(HaveKey("SAML_PRIVATE_KEY_FILENAME"))

		err = ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.saml", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchSecret.Data).To(HaveKeyWithValue("sp.crt", []byte("veryverifiedcert")))
	})

	It("handles a SAML_IDP_PUBLIC_KEY", func() {
		inSecret.Data["SAML_IDP_PUBLIC_KEY"] = []byte("veryverifiedcert")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		appSecretsData := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &appSecretsData)
		Expect(err).ToNot(HaveOccurred())
		Expect(appSecretsData).To(HaveKeyWithValue("SAML_IDP_PUBLIC_KEY_FILENAME", "idp.crt"))

		err = ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.saml", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchSecret.Data).To(HaveKeyWithValue("idp.crt", []byte("veryverifiedcert")))
	})

	It("handles a SAML_IDP_METADATA", func() {
		inSecret.Data["SAML_IDP_METADATA"] = []byte("<saml>isgreat</saml>")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.app-secrets", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		appSecretsData := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["summon-platform.yml"], &appSecretsData)
		Expect(err).ToNot(HaveOccurred())
		Expect(appSecretsData).To(HaveKeyWithValue("SAML_IDP_METADATA_FILENAME", "metadata.xml"))
		Expect(appSecretsData).To(HaveKeyWithValue("SAML_USE_LOCAL_METADATA", true))

		err = ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.saml", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchSecret.Data).To(HaveKeyWithValue("metadata.xml", []byte("<saml>isgreat</saml>")))
	})

	It("creates a comp-dispatch secret", func() {
		inSecret.Data["GOOGLE_MAPS_BACKEND_API_KEY"] = []byte("asdf1234")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.dispatch", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		data := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["dispatch.yml"], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("Debug", false))
		Expect(data).To(HaveKeyWithValue("DatabaseURL", "postgres://foo_dev:postgresPassword@summon-dev-database/foo_dev"))
		Expect(data).To(HaveKeyWithValue("GoogleApiKey", "asdf1234"))
	})

	It("sets comp-dispatch to debug if summon is DEBUG=True", func() {
		v := true
		instance.Spec.Config = map[string]summonv1beta1.ConfigValue{
			"DEBUG": summonv1beta1.ConfigValue{Bool: &v},
		}
		inSecret.Data["GOOGLE_MAPS_BACKEND_API_KEY"] = []byte("asdf1234")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.dispatch", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		data := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["dispatch.yml"], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("Debug", true))
	})

	It("creates a comp-hw-aux secret", func() {
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.hwaux", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		data := map[string]interface{}{}
		err = yaml.Unmarshal(fetchSecret.Data["hwaux.yml"], &data)
		Expect(err).ToNot(HaveOccurred())
	})

	It("creates a comp-trip-share secret", func() {
		inSecret.Data["GOOGLE_MAPS_BACKEND_API_KEY"] = []byte("asdf1234")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.tripshare", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		data := map[string]interface{}{}
		err = json.Unmarshal(fetchSecret.Data["config.json"], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("google_api_key", "asdf1234"))
	})

	It("creates a comp-trip-share secret with a tripshare specific key", func() {
		inSecret.Data["GOOGLE_MAPS_BACKEND_API_KEY"] = []byte("asdf1234")
		inSecret.Data["GOOGLE_MAPS_TRIPSHARE_API_KEY"] = []byte("qwer5678")
		ctx.Client = fake.NewFakeClient(inSecret, postgresSecret, fernetKeys, secretKey, accessKey, rabbitmqPassword)
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "foo-dev.tripshare", Namespace: "summon-dev"}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())

		data := map[string]interface{}{}
		err = json.Unmarshal(fetchSecret.Data["config.json"], &data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("google_api_key", "qwer5678"))
	})

})
