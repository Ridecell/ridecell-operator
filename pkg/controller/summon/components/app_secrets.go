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
	"strings"
	"time"

	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
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
		for _, secret := range append(summon.Spec.Secrets, comp.inputSecrets(&summon)...) {
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

	if instance.Status.PostgresStatus != postgresv1.ClusterStatusRunning {
		return false
	}

	return true
}

func (comp *appSecretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	specInputSecrets, err := comp.fetchSecrets(ctx, instance, instance.Spec.Secrets, false)
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
	rabbitmqPassword := dynamicInputSecrets[4]

	fmt.Println(rabbitmqPassword)
	postgresDatabase, postgresUser := comp.postgresNames(instance)
	postgresPassword, ok := postgresSecret.Data["password"]
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
		return components.Result{Requeue: true}, errors.Errorf("app_secrets: Invalid data in SECRET_KEY secret: %s", val)
	}

	appSecretsData := map[string]interface{}{}

	appSecretsData["DATABASE_URL"] = fmt.Sprintf("postgis://%s:%s@%s-database/%s", postgresUser, postgresPassword, postgresDatabase, postgresUser)
	appSecretsData["OUTBOUNDSMS_URL"] = fmt.Sprintf("https://%s.prod.ridecell.io/outbound-sms", instance.Name)
	appSecretsData["SMS_WEBHOOK_URL"] = fmt.Sprintf("https://%s.ridecell.us/sms/receive/", instance.Name)
	appSecretsData["CELERY_BROKER_URL"] = fmt.Sprintf("redis://%s-redis/2", instance.Name)
	appSecretsData["FERNET_KEYS"] = formattedFernetKeys
	appSecretsData["SECRET_KEY"] = string(secretKey.Data["SECRET_KEY"])
	appSecretsData["AWS_ACCESS_KEY_ID"] = string(awsSecret.Data["AWS_ACCESS_KEY_ID"])
	appSecretsData["AWS_SECRET_ACCESS_KEY"] = string(awsSecret.Data["AWS_SECRET_ACCESS_KEY"])

	for _, secret := range specInputSecrets {
		for k, v := range secret.Data {
			appSecretsData[k] = string(v)
		}
	}

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
		return components.Result{}, errors.Wrapf(err, "app_secrets: Failed to update secret object")
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

func (_ *appSecretComponent) postgresNames(instance *summonv1beta1.SummonPlatform) (string, string) {
	if instance.Spec.Database.ExclusiveDatabase {
		return instance.Name, "summon"
	} else {
		return instance.Spec.Database.SharedDatabaseName, strings.Replace(instance.Name, "-", "_", -1)
	}
}

func (c *appSecretComponent) inputSecrets(instance *summonv1beta1.SummonPlatform) []string {
	postgresDatabase, postgresUser := c.postgresNames(instance)

	// The order of these must match the code using it. Do not change. I mean it.
	return []string{
		fmt.Sprintf("%s.%s-database.credentials", strings.Replace(postgresUser, "_", "-", -1), postgresDatabase),
		fmt.Sprintf("%s.fernet-keys", instance.Name),
		fmt.Sprintf("%s.secret-key", instance.Name),
		fmt.Sprintf("%s.aws-credentials", instance.Name),
		fmt.Sprintf("%s.rabbitmq-user-password", instance.Name),
	}
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
