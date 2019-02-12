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
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (_ *appSecretComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	if instance.Status.PostgresStatus != postgresv1.ClusterStatusRunning {
		return false
	}
	return true
}

func (comp *appSecretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	var rawAppSecrets []*corev1.Secret
	for _, secretName := range instance.Spec.Secrets {
		rawAppSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: secretName, Namespace: instance.Namespace}, rawAppSecret)
		if err != nil {
			return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: Failed to get input app secrets %s", secretName)
		}
		rawAppSecrets = append(rawAppSecrets, rawAppSecret)
	}

	databaseName := instance.Spec.Database.SharedDatabaseName
	databaseUser := strings.Replace(instance.Name, "-", "_", -1)
	if instance.Spec.Database.ExclusiveDatabase {
		databaseName = instance.Name
		databaseUser = "summon"
	}

	postgresSecret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.%s-database.credentials", databaseUser, databaseName), Namespace: instance.Namespace}, postgresSecret)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Don't trigger an error on notfound so it doesn't notify. Just try again.
			return components.Result{Requeue: true}, nil
		} else {
			return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: Postgres password not found")
		}
	}
	postgresPassword, ok := postgresSecret.Data["password"]
	if !ok {
		return components.Result{}, errors.New("app_secrets: Postgres password not found in secret")
	}

	fernetKeys := &corev1.Secret{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.fernet-keys", instance.Name), Namespace: instance.Namespace}, fernetKeys)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Don't trigger an error on notfound so it doesn't notify. Just try again.
			return components.Result{Requeue: true}, nil
		} else {
			return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: Fernet keys secret not found")
		}
	}
	if len(fernetKeys.Data) == 0 {
		return components.Result{}, errors.New("app_secrets: Fernet keys map is empty")
	}

	formattedFernetKeys, err := comp.formatFernetKeys(fernetKeys.Data)
	if err != nil {
		return components.Result{}, err
	}

	secretKey := &corev1.Secret{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.secret-key", instance.Name), Namespace: instance.Namespace}, secretKey)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Don't trigger an error on notfound so it doesn't notify. Just try again.
			return components.Result{Requeue: true}, nil
		} else {
			return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: Unable to get SECRET_KEY")
		}
	}
	val, ok := secretKey.Data["SECRET_KEY"]
	if !ok || len(val) == 0 {
		return components.Result{Requeue: true}, errors.Errorf("app_secrets: Invalid data in SECRET_KEY secret: %s", val)
	}

	accessKey := &corev1.Secret{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.aws-credentials", instance.Name), Namespace: instance.Namespace}, accessKey)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Don't trigger an error on notfound so it doesn't notify. Just try again.
			return components.Result{Requeue: true}, nil
		} else {
			return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: Unable to get aws credentials secret")
		}
	}

	appSecretsData := map[string]interface{}{}

	appSecretsData["DATABASE_URL"] = fmt.Sprintf("postgis://%s:%s@%s-database/%s", databaseUser, postgresPassword, databaseName, databaseUser)
	appSecretsData["OUTBOUNDSMS_URL"] = fmt.Sprintf("https://%s.prod.ridecell.io/outbound-sms", instance.Name)
	appSecretsData["SMS_WEBHOOK_URL"] = fmt.Sprintf("https://%s.ridecell.us/sms/receive/", instance.Name)
	appSecretsData["CELERY_BROKER_URL"] = fmt.Sprintf("redis://%s-redis/2", instance.Name)
	appSecretsData["FERNET_KEYS"] = formattedFernetKeys
	appSecretsData["SECRET_KEY"] = string(secretKey.Data["SECRET_KEY"])
	appSecretsData["AWS_ACCESS_KEY_ID"] = string(accessKey.Data["AWS_SECRET_ACCESS_KEY"])
	appSecretsData["AWS_SECRET_ACCESS_KEY"] = string(accessKey.Data["AWS_SECRET_ACCESS_KEY"])

	for _, rawAppSecretObj := range rawAppSecrets {
		for k, v := range rawAppSecretObj.Data {
			appSecretsData[k] = string(v)
		}
	}

	parsedYaml, err := yaml.Marshal(appSecretsData)
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "app_secrets: yaml.Marshal failed")
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("summon.%s.app-secrets", instance.Name), Namespace: instance.Namespace},
		Data:       map[string][]byte{"summon-platform.yml": parsedYaml},
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
