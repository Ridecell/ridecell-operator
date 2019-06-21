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

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	alertconfig "github.com/prometheus/alertmanager/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type alertManageConfigComponent struct {
}

func NewAlertManagerConfig() *alertManageConfigComponent {
	return &alertManageConfigComponent{}
}

func (_ *alertManageConfigComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *alertManageConfigComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *alertManageConfigComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	// get default alertmanger config
	instance := ctx.Top.(*monitoringv1beta1.AlertManagerConfig)
	defaultConfigSecret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{
		Name:      fmt.Sprintf("%s-default", instance.Spec.AlertManagerName),
		Namespace: instance.Spec.AlertManagerNamespace}, defaultConfigSecret)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to get default AlertManagerConfig")
	}
	// verify default config
	defaultConfig, err := alertconfig.Load(string(defaultConfigSecret.Data["alertmanager.yaml"][:]))
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Looks like default config in bad format.")
	}
	// Get all AlertManagerConfig's
	alertList := &monitoringv1beta1.AlertManagerConfigList{}
	err = ctx.List(ctx.Context, &client.ListOptions{}, alertList)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "failed to list alermanagerconfig.")
	}
	// Merge  all AlertManagerConfig's
	for _, config := range alertList.Items {
		routetype := &alertconfig.Route{}
		receivertype := &alertconfig.Receiver{}
		inhibitRuletype := &alertconfig.InhibitRule{}
		if config.Spec.AlertManagerName == instance.Spec.AlertManagerName {
			errRo := yaml.Unmarshal([]byte(config.Spec.Data["routes"]), routetype)
			errRe := yaml.Unmarshal([]byte(config.Spec.Data["receiver"]), receivertype)
			errIn := yaml.Unmarshal([]byte(config.Spec.Data["receiver"]), inhibitRuletype)
			if errRo != nil || errRe != nil || errIn != nil {
				glog.Errorf("failed to load yaml for %s in %s", config.Name, config.Namespace)
				if instance.UID == config.UID {
					return components.Result{}, errors.Errorf("failed to load yaml for %s in %s", config.Name, config.Namespace)
				}
				continue
			}
			defaultConfig.Route.Routes = append(defaultConfig.Route.Routes, routetype)
			defaultConfig.Receivers = append(defaultConfig.Receivers, receivertype)
			defaultConfig.InhibitRules = append(defaultConfig.InhibitRules, inhibitRuletype)
		}
	}
	// Merged config
	finalConfig, _ := yaml.Marshal(defaultConfig)
	// verify config
	_, err = alertconfig.Load(string(finalConfig))
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to verify alertmanger config")
	}

	// Create/Update secret with finalConfig which prometheus-operator can attach to alertmanager
	// https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/alerting.md
	// prometheus-operator need alertconfig as  kind  secret with format check above link for more info
	alertConfigFinal := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("alertmanager-%s", instance.Spec.AlertManagerName),
			Namespace: instance.Spec.AlertManagerNamespace,
		},
	}
	// Get attached config and overwrite with default config
	err = ctx.Get(ctx.Context, types.NamespacedName{
		Namespace: instance.Spec.AlertManagerNamespace,
		Name:      fmt.Sprintf("alertmanager-%s", instance.Spec.AlertManagerName),
	}, alertConfigFinal)
	if err != nil {
		glog.Infof("creating config as secret for %s", instance.Spec.AlertManagerName)
		alertConfigFinal.Data = map[string][]byte{"alertmanager.yml": []byte(finalConfig)}
		// Remove "alertmanager.yaml" key from DefaultData  adding rest DefaultData. So we can keep rest of the templates.
		delete(defaultConfigSecret.Data, "alertmanager.yaml")
		for k, v := range defaultConfigSecret.Data {
			alertConfigFinal.Data[k] = v
		}
		err = ctx.Create(ctx.Context, alertConfigFinal)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "Failed to create secret as config for %s", instance.Spec.AlertManagerName)
		}
		return components.Result{}, nil
	}
	// Update secret with finalconfig
	alertConfigFinal.Data = map[string][]byte{"alertmanager.yml": []byte(finalConfig)}
	// Remove "alertmanager.yaml" key from DefaultData  adding rest DefaultData. So we can keep rest of the templates.
	delete(defaultConfigSecret.Data, "alertmanager.yaml")
	for k, v := range defaultConfigSecret.Data {
		alertConfigFinal.Data[k] = v
	}
	err = ctx.Update(ctx.Context, alertConfigFinal)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to update secret as config for %s", instance.Spec.AlertManagerName)
	}
	return components.Result{}, nil
}
