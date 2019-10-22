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

package components

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

type appSecretComponent struct{}

type fernetKeyEntry struct {
	Key  []byte
	Date time.Time
}

type fernetSlice []fernetKeyEntry

func (p fernetSlice) Len() int {
	return len(p)
}

func (p fernetSlice) Less(i, j int) bool {
	return p[i].Date.Before(p[j].Date)
}

func (p fernetSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func NewAppSecret() *appSecretComponent {
	return &appSecretComponent{}
}

func (comp *appSecretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
		&dbv1beta1.RabbitmqVhost{},
	}
}

func (comp *appSecretComponent) WatchMap(obj handler.MapObject, c client.Client) ([]reconcile.Request, error) {
	// First check if this is an owned object, if so, short circuit.
	owner := metav1.GetControllerOf(obj.Meta)
	if owner != nil && owner.Kind == "SummonPlatform" {
		return []reconcile.Request{
			reconcile.Request{NamespacedName: types.NamespacedName{Name: owner.Name, Namespace: obj.Meta.GetNamespace()}},
		}, nil
	}

	// Search all SummonPlatforms to see if any mention this as an input secret.
	summons := &summonv1beta1.SummonPlatformList{}
	err := c.List(context.Background(), nil, summons)
	if err != nil {
		return nil, errors.Wrap(err, "error listing summonplatforms")
	}

	requests := []reconcile.Request{}
	for _, summon := range summons.Items {
		// TODO Can do this with a list option once that API stabilizes
		if summon.Namespace != obj.Meta.GetNamespace() {
			continue
		}

		// Check the input secrets.
		for _, secret := range append(comp.specSecrets(&summon), comp.inputSecrets(&summon)...) {
			if obj.Meta.GetName() == secret {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: summon.Name, Namespace: summon.Namespace}})
				break
			}
		}
	}

	return requests, nil
}

func (_ *appSecretComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	if instance.Status.PostgresStatus != dbv1beta1.StatusReady {
		return false
	}

	if instance.Status.RabbitMQStatus != dbv1beta1.StatusReady {
		return false
	}

	return true
}

func (comp *appSecretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	specInputSecrets, err := comp.fetchSecrets(ctx, instance, comp.specSecrets(instance), false)
	if err != nil {
		return components.Result{}, err
	}
	dynamicInputSecrets, err := comp.fetchSecrets(ctx, instance, comp.inputSecrets(instance), true)
	if err != nil {
		return components.Result{Requeue: true}, err
	}

	// This order must match the one in inputSecrets().
	postgresSecret := dynamicInputSecrets[0]
	fernetKeys := dynamicInputSecrets[1]
	secretKey := dynamicInputSecrets[2]
	awsSecret := dynamicInputSecrets[3]
	rabbitmqSecret := dynamicInputSecrets[4]

	postgresConnection := instance.Status.PostgresConnection
	postgresPassword, ok := postgresSecret.Data[postgresConnection.PasswordSecretRef.Key]
	if !ok {
		return components.Result{}, errors.New("app_secrets: Postgres password not found in secret")
	}

	if len(fernetKeys.Data) == 0 {
		return components.Result{}, errors.New("app_secrets: Fernet keys map is empty")
	}

	formattedFernetKeys, err := comp.formatFernetKeys(fernetKeys.Data)
	if err != nil {
		return components.Result{}, err
	}

	val, ok := secretKey.Data["SECRET_KEY"]
	if !ok || len(val) == 0 {
		return components.Result{}, errors.Errorf("app_secrets: Invalid data in SECRET_KEY secret: %s", val)
	}

	// Find the RabbitMQ data.
	rabbitmqConnection := instance.Status.RabbitMQConnection
	rabbitmqPassword, ok := rabbitmqSecret.Data[rabbitmqConnection.PasswordSecretRef.Key]
	if !ok {
		return components.Result{}, errors.Errorf("app_secrets: RabbitMQ password not found in secret %s[%s]", rabbitmqConnection.PasswordSecretRef.Name, rabbitmqConnection.PasswordSecretRef.Key)
	}

	appSecretsData := map[string]interface{}{}

	// Set up dynamic-y values.
	appSecretsData["DATABASE_URL"] = fmt.Sprintf("postgis://%s:%s@%s/%s", postgresConnection.Username, postgresPassword, postgresConnection.Host, postgresConnection.Database)
	appSecretsData["OUTBOUNDSMS_URL"] = fmt.Sprintf("https://%s.prod.ridecell.io/outbound-sms", instance.Name)
	appSecretsData["SMS_WEBHOOK_URL"] = fmt.Sprintf("https://%s.ridecell.us/sms/receive/", instance.Name)
	appSecretsData["CELERY_BROKER_URL"] = fmt.Sprintf("pyamqp://%s:%s@%s/%s?ssl=true", rabbitmqConnection.Username, rabbitmqPassword, rabbitmqConnection.Host, rabbitmqConnection.Vhost)
	appSecretsData["FERNET_KEYS"] = formattedFernetKeys
	appSecretsData["SECRET_KEY"] = string(secretKey.Data["SECRET_KEY"])
	appSecretsData["AWS_ACCESS_KEY_ID"] = string(awsSecret.Data["AWS_ACCESS_KEY_ID"])
	appSecretsData["AWS_SECRET_ACCESS_KEY"] = string(awsSecret.Data["AWS_SECRET_ACCESS_KEY"])

	// Insert input secret overrides in the correct order.
	for _, secret := range specInputSecrets {
		for k, v := range secret.Data {
			appSecretsData[k] = string(v)
		}
	}

	// If OTAKEYS_API_KEY is provided externally and EnableMockCarServer is also true, it is a conflict
	v := appSecretsData["OTAKEYS_API_KEY"]
	if (v != nil && len(v.(string)) > 0) && instance.Spec.EnableMockCarServer {
		return components.Result{}, errors.Errorf("app_secrets: Conflict in OTA Keys configuration")
	}
	if instance.Spec.EnableMockCarServer {
		otaSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name + ".tenant-otakeys", Namespace: instance.Namespace}, otaSecret)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "app_secrets: Unable to fetch otakeys secret")
		}
		for k, v := range otaSecret.Data {
			appSecretsData[k] = string(v)
		}
	}

	// Pull out specific keys into the secondary SAML-specific secret.
	samlSecretsData := map[string][]byte{}
	privateKey, ok := appSecretsData["SAML_PRIVATE_KEY"]
	if ok {
		samlSecretsData["sp.key"] = []byte(privateKey.(string))
		appSecretsData["SAML_PRIVATE_KEY_FILENAME"] = "sp.key"
	}
	publicKey, ok := appSecretsData["SAML_PUBLIC_KEY"]
	if ok {
		samlSecretsData["sp.crt"] = []byte(publicKey.(string))
		appSecretsData["SAML_PUBLIC_KEY_FILENAME"] = "sp.crt"
	}
	idpPublicKey, ok := appSecretsData["SAML_IDP_PUBLIC_KEY"]
	if ok {
		samlSecretsData["idp.crt"] = []byte(idpPublicKey.(string))
		appSecretsData["SAML_IDP_PUBLIC_KEY_FILENAME"] = "idp.crt"
	}
	idpMetadata, ok := appSecretsData["SAML_IDP_METADATA"]
	if ok {
		samlSecretsData["metadata.xml"] = []byte(idpMetadata.(string))
		appSecretsData["SAML_IDP_METADATA_FILENAME"] = "metadata.xml"
		// If we set local metadata, we probably always want to use it. Should help avoid double-config drift.
		appSecretsData["SAML_USE_LOCAL_METADATA"] = true
	}

	// Serialize the app secrets YAML and put it in a secret.
	yamlData, err := yaml.Marshal(appSecretsData)
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: yaml.Marshal failed")
	}
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
		Data:       map[string][]byte{"summon-platform.yml": yamlData},
	}

	_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, newSecret.DeepCopy(), func(existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		// Sync important fields.
		err := controllerutil.SetControllerReference(instance, existing, ctx.Scheme)
		if err != nil {
			return errors.Wrapf(err, "app_secrets: Failed to set controller reference")
		}
		existing.Labels = newSecret.Labels
		existing.Annotations = newSecret.Annotations
		existing.Type = newSecret.Type
		existing.Data = newSecret.Data
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "app_secrets: Failed to update app secret object")
	}

	// Create the SAML secret, even if it's empty as it usually will be.
	_, _, err = ctx.CreateOrUpdate("secrets/saml.yml.tpl", nil, func(_, existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		existing.Data = samlSecretsData
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "app_secrets: Failed to update SAML secret object")
	}

	return components.Result{}, nil
}

func (_ *appSecretComponent) formatFernetKeys(fernetData map[string][]byte) ([]string, error) {
	var unsortedArray []fernetKeyEntry
	for k, v := range fernetData {
		parsedTime, err := time.Parse(CustomTimeLayout, k)
		if err != nil {
			return nil, errors.New("app_secrets: Failed to parse time for fernet keys")
		}
		unsortedArray = append(unsortedArray, fernetKeyEntry{Date: parsedTime, Key: v})
	}

	sortedTimes := make(fernetSlice, 0, len(unsortedArray))
	for _, d := range unsortedArray {
		sortedTimes = append(sortedTimes, d)
	}

	sort.Sort(sort.Reverse(sortedTimes))

	var outputSlice []string
	for _, v := range sortedTimes {
		outputSlice = append(outputSlice, string(v.Key))
	}

	return outputSlice, nil
}

func (c *appSecretComponent) inputSecrets(instance *summonv1beta1.SummonPlatform) []string {
	// The order of these must match the code using it. Do not change. I mean it.
	return []string{
		instance.Status.PostgresConnection.PasswordSecretRef.Name,
		fmt.Sprintf("%s.fernet-keys", instance.Name),
		fmt.Sprintf("%s.secret-key", instance.Name),
		fmt.Sprintf("%s.aws-credentials", instance.Name),
		instance.Status.RabbitMQConnection.PasswordSecretRef.Name,
	}
}

func (c *appSecretComponent) specSecrets(instance *summonv1beta1.SummonPlatform) []string {
	if len(instance.Spec.Secrets) == 0 {
		return []string{instance.Namespace, instance.Name}
	}
	return instance.Spec.Secrets
}

func (_ *appSecretComponent) fetchSecrets(ctx *components.ComponentContext, instance *summonv1beta1.SummonPlatform, secretNames []string, allowMissing bool) ([]*corev1.Secret, error) {
	secrets := []*corev1.Secret{}
	for _, secretName := range secretNames {
		secret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: secretName, Namespace: instance.Namespace}, secret)
		if err != nil {
			if kerrors.IsNotFound(err) && allowMissing {
				err = errors.NoNotify(err)
			}
			label := "input"
			if allowMissing {
				label = "derived"
			}
			return nil, errors.Wrapf(err, "app_secrets: error fetching %s app secret %s", label, secretName)
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}
