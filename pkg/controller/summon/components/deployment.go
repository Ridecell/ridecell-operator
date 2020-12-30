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
	"strings"

	kedav1alpha1 "github.com/Ridecell/ridecell-operator/pkg/apis/keda/v1alpha1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type deploymentComponent struct {
	templatePath string
	isAutoscaled func(*summonv1beta1.SummonPlatform) bool
}

func NewDeployment(templatePath string, isAutoscaled func(*summonv1beta1.SummonPlatform) bool) *deploymentComponent {
	return &deploymentComponent{templatePath: templatePath, isAutoscaled: isAutoscaled}
}

func (comp *deploymentComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
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

	// If we're not in deploying state do nothing and exit early.
	if instance.Status.Status != summonv1beta1.StatusDeploying {
		return components.Result{}, nil
	}

	// Create either Celery beat or Celery RedBeat
	if strings.HasPrefix(comp.templatePath, "celerybeat") && instance.Spec.UseCeleryRedBeat {
		// delete celerybeat
		return components.Result{}, comp.deleteObject(ctx, instance, "celerybeat")
	} else if strings.HasPrefix(comp.templatePath, "celeryredbeat") && !(instance.Spec.UseCeleryRedBeat) {
		// delete celeryredbeat
		return components.Result{}, comp.deleteObject(ctx, instance, "celeryredbeat")
	}

	// TODO 2020-01-06 After cm+secret merges to just secret, support varying the input names in the component config so comp-dispatch and comp-trip-share can get just the hash of their config.
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
		return components.Result{}, errors.Wrapf(err, "deployment: unable to serialize appsecrets")
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
	// Pass debug value
	_, ok := instance.Spec.Config["DEBUG"]
	extra["debug"] = bool(ok)

	// Autoscaling check
	if comp.isAutoscaled != nil && comp.isAutoscaled(instance) {
		// Create scaled object
		templatePath := fmt.Sprintf("%s/scaledobject.yml", strings.Split(comp.templatePath, "/")[0])
		res, _, err := ctx.CreateOrUpdate(templatePath, nil, func(goalObj, existingObj runtime.Object) error {
			goalDeployment, ok := goalObj.(*kedav1alpha1.ScaledObject)
			existing := existingObj.(*kedav1alpha1.ScaledObject)
			// check if any custom scaledobject is already present for same deployment, if true, then make no change
			if existing.Annotations != nil && existing.Annotations["ridecell.io/custom-object"] == "true" {
				return nil
			}
			existing.Spec = goalDeployment.Spec
			return nil
		})
		if err != nil {
			return res, errors.Wrapf(err, "deployment: failed to update template %s", templatePath)
		}
	} else {
		// Delete scaled object if exists
		_, err := comp.deleteObject(ctx, instance, "scaledobject")
		if err != nil {
			return components.Result{}, err
		}
	}

	res, _, err := ctx.CreateOrUpdate(comp.templatePath, extra, func(goalObj, existingObj runtime.Object) error {
		goalDeployment, ok := goalObj.(*appsv1.Deployment)
		if ok {
			existing := existingObj.(*appsv1.Deployment)
			// Check if autoscaling was enabled and keep existing deployment replicas setting set by HPA
			if comp.isAutoscaled != nil && comp.isAutoscaled(instance) {
				goalDeployment.Spec.Replicas = existing.Spec.Replicas
			}
			existing.Spec = goalDeployment.Spec
			return nil
		}

		goalStatefulSet := goalObj.(*appsv1.StatefulSet)
		existing := existingObj.(*appsv1.StatefulSet)
		existing.Spec = goalStatefulSet.Spec
		return nil
	})
	if err != nil {
		return res, errors.Wrapf(err, "deployment: failed to update template %s", comp.templatePath)
	}
	return components.Result{}, nil
}

func (_ *deploymentComponent) hashItem(data []byte) string {
	hash := sha1.Sum(data)
	encodedHash := hex.EncodeToString(hash[:])
	return encodedHash
}

func (_ *deploymentComponent) deleteObject(ctx *components.ComponentContext, instance *summonv1beta1.SummonPlatform, componentName string) error {
	var obj runtime.Object
	if componentName == "celerybeat" {
		obj = &appsv1.StatefulSet{}
	} else if componentName == "scaledobject" {
		obj = &kedav1alpha1.ScaledObject{}
	} else {
		obj = &appsv1.Deployment{}
	}

	err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: instance.Name + "-" + componentName, Namespace: instance.Namespace}, obj)
	if err == nil {
		err = ctx.Delete(ctx.Context, obj)
		if err != nil {
			return errors.Wrapf(err, "deployment: failed to delete existing "+componentName)
		}
	} else if err != nil && !k8serrors.IsNotFound(err) {
		return errors.Wrapf(err, "deployment: failed to get and delete existing "+componentName)
	}
	return nil
}
