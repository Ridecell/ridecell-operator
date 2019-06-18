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
	"encoding/base64"
	"fmt"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	alertmconfig "github.com/prometheus/alertmanager/config"
)

type notificationComponent struct {
}

func Newnotification() *notificationComponent {
	return &notificationComponent{}
}

func (_ *notificationComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *notificationComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *notificationComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*monitoringv1beta1.Monitor)
	// slack config
	if len(instance.Spec.Notify.Slack) <= 0 {
		return components.Result{}, errors.New(fmt.Sprintf("No slack chanel defined  for %s", instance.Name))
	}
	slackconfigs := []*alertmconfig.SlackConfig{}
	for _, channel := range instance.Spec.Notify.Slack {
		// add chanel
		slackconfigs = append(slackconfigs, &alertmconfig.SlackConfig{
			NotifierConfig: alertmconfig.NotifierConfig{
				VSendResolved: false,
			},
			Channel:   channel,
			Title:     `'{{ template "slack.ridecell.title" . }}'`,
			IconEmoji: `'{{ template "slack.ridecell.icon_emoji" . }}'`,
			Color:     `'{{ template "slack.ridecell.color" . }}'`,
			Text:      `'{{ template "slack.ridecell.text" . }}'`})
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
	extras["routes"] = base64.StdEncoding.EncodeToString(marshled)
	marshled, _ = yaml.Marshal(receiver)
	extras["receiver"] = base64.StdEncoding.EncodeToString(marshled)

	res, _, err := ctx.CreateOrUpdate("alertmanagerconfig.yml.tpl", extras, func(_goalObj, existingObj runtime.Object) error {
		_ = existingObj.(*monitoringv1beta1.AlertManagerConfig)
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to create AlertManagerConfig for %s", instance.Name)
	}
	return res, nil
}
