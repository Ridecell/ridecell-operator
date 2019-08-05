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

package components

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type deploymentComponent struct {
	templatePath string
}

func NewDeployment(templatePath string) *deploymentComponent {
	return &deploymentComponent{templatePath: templatePath}
}

func (comp *deploymentComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&appsv1.Deployment{},
	}
}

func (comp *deploymentComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// Check on the pull secret. Not technically needed in some cases, but just wait.
	if instance.Status.PullSecretStatus != secretsv1beta1.StatusReady {
		return false
	}
	// We do want the database, so check all the database statuses.
	if instance.Status.PostgresStatus != dbv1beta1.StatusReady {
		return false
	}
	if instance.Status.Status == summonv1beta1.StatusReady {
		return true
	}
	if instance.Status.Status != summonv1beta1.StatusDeploying {
		return false
	}
	return true
}

func (comp *deploymentComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	rawAppSecret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace}, rawAppSecret)
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "deployment: Failed to get appsecrets")
	}

	config := &corev1.ConfigMap{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace}, config)
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "deployment: unable to get configmap")
	}

	appSecretsBytes, err := json.Marshal(rawAppSecret.Data)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "deployment: unable to serialize appsecrets ")
	}
	configBytes, err := json.Marshal(config.Data)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "deployment: unable to serialize config")
	}

	appSecretsHash := comp.hashItem(appSecretsBytes)
	configMapHash := comp.hashItem(configBytes)

	// Data to be copied over to template
	extra := map[string]interface{}{}
	extra["configHash"] = string(configMapHash)
	extra["appSecretsHash"] = string(appSecretsHash)

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, extra, func(goalObj, existingObj runtime.Object) error {
		goalDeployment, ok := goalObj.(*appsv1.Deployment)
		if ok {
			existing := existingObj.(*appsv1.Deployment)
			existing.Spec = goalDeployment.Spec
			return nil
		}

		goalStatefulSet := goalObj.(*appsv1.StatefulSet)
		existing := goalObj.(*appsv1.StatefulSet)
		existing.Spec = goalStatefulSet.Spec
		return nil
	})
	if err != nil {
		return res, errors.Wrapf(err, "deployment: failed to update template")
	}
	return components.Result{}, nil
}

func (_ *deploymentComponent) hashItem(data []byte) string {
	hash := sha1.Sum(data)
	encodedHash := hex.EncodeToString(hash[:])
	return encodedHash
}
