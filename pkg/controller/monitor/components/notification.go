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
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"

	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	alertmconfig "github.com/prometheus/alertmanager/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const notificationFinalizer = "finalizer.notification.monitoring.ridecell.io"

type notificationComponent struct {
}

func NewNotification() *notificationComponent {
	return &notificationComponent{}
}

func (_ *notificationComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&monitoringv1beta1.Monitor{},
	}
}

func (_ *notificationComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *notificationComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*monitoringv1beta1.Monitor)

	// slack config
	if len(instance.Spec.Notify.Slack) <= 0 {
		//return components.Result{}, errors.Errorf("No slack chanel defined  for %s", instance.Name)
		return components.Result{}, nil
	}

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(notificationFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(notificationFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(notificationFinalizer, instance) {
			promrule := &monitoringv1beta1.AlertManagerConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("alertmanagerconfig-%s", instance.Name),
					Namespace: instance.Namespace,
				}}
			err := ctx.Delete(ctx.Context, promrule)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to delete notification %s", instance.Name)
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(notificationFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update notification  while removing finalizer")
			}
		}
		return components.Result{}, nil
	}

	slackconfigs := []*alertmconfig.SlackConfig{}
	for _, channel := range instance.Spec.Notify.Slack {
		// add chanel
		slackconfigs = append(slackconfigs, &alertmconfig.SlackConfig{
			NotifierConfig: alertmconfig.NotifierConfig{
				VSendResolved: false,
			},
			Channel:   channel,
			Title:     `{{ template "slack.ridecell.title" . }}`,
			IconEmoji: `{{ template "slack.ridecell.icon_emoji" . }}`,
			Color:     `{{ template "slack.ridecell.color" . }}`,
			Text:      `{{ template "slack.ridecell.text" . }}`})
	}
	// Create receiver
	receiver := &alertmconfig.Receiver{
		Name:         instance.Name,
		SlackConfigs: slackconfigs,
	}
	// Create route and group by namspace
	routes := &alertmconfig.Route{
		Receiver:   instance.Name,
		Continue:   true,
		GroupByStr: []string{"namespace"},
		Match: map[string]string{
			"namespace": instance.Namespace,
		},
	}

	extras := map[string]interface{}{}
	marshled, _ := yaml.Marshal(routes)
	extras["routes"] = string(marshled)
	marshled, _ = yaml.Marshal(receiver)
	extras["receiver"] = string(marshled)

	res, _, err := ctx.CreateOrUpdate("alertmanagerconfig.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*monitoringv1beta1.AlertManagerConfig)
		existing := existingObj.(*monitoringv1beta1.AlertManagerConfig)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to create AlertManagerConfig for %s", instance.Name)
	}
	return res, nil
}
