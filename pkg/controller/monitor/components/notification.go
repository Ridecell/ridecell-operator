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
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	pagerduty "github.com/heimweh/go-pagerduty/pagerduty"
	alertmconfig "github.com/prometheus/alertmanager/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const notificationFinalizer = "finalizer.notification.monitoring.ridecell.io"

type notificationComponent struct {
	PgBaseURL string
}

func NewNotification() *notificationComponent {
	return &notificationComponent{}
}

func (comp *notificationComponent) UpdateBaseURL(pgBaseURL string) {
	comp.PgBaseURL = pgBaseURL
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
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, instance)
	if err != nil {
		// Error reading the object - requeue the request.
		return components.Result{}, errors.Wrapf(err, "instance of ridecellingress not found")
	}
	// slack config
	if len(instance.Spec.Notify.Slack) <= 0 && len(instance.Spec.Notify.PagerdutyTeam) <= 0 {
		//return components.Result{}, errors.Errorf("No slack chanel defined  for %s", instance.Name)
		return components.Result{}, nil
	}

	client, _ := pagerduty.NewClient(&pagerduty.Config{Token: os.Getenv("PG_API_KEY"), BaseURL: "https://api.pagerduty.com"})
	if len(os.Getenv("PG_MOCK_URL")) > 0 {
		client, _ = pagerduty.NewClient(&pagerduty.Config{Token: os.Getenv("PG_API_KEY"), BaseURL: os.Getenv("PG_MOCK_URL")})
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
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" && os.Getenv("ENABLE_FINALIZERS") == "true" {
				//remove alertmanagrconfig
				amc := &monitoringv1beta1.AlertManagerConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("alertmanagerconfig-%s", instance.Name),
						Namespace: instance.Namespace,
					}}
				err := ctx.Delete(ctx.Context, amc)
				if err != nil {
					return components.Result{}, errors.Wrapf(err, "failed to delete notification %s", instance.Name)
				}
				// TODO remove service/event rule from PG
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(notificationFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update notification while removing finalizer")
			}
		}
		return components.Result{}, nil
	}

	extras := map[string]interface{}{}

	if len(instance.Spec.Notify.Slack) > 0 {
		receiverSlack := &alertmconfig.Receiver{
			Name:         instance.Name + "-slack",
			SlackConfigs: []*alertmconfig.SlackConfig{},
		}
		for _, channel := range instance.Spec.Notify.Slack {
			// add chanel
			receiverSlack.SlackConfigs = append(receiverSlack.SlackConfigs, &alertmconfig.SlackConfig{
				NotifierConfig: alertmconfig.NotifierConfig{
					VSendResolved: true,
				},
				Channel:   channel,
				Title:     `{{ template "slack.ridecell.title" . }}`,
				IconEmoji: `{{ template "slack.ridecell.icon_emoji" . }}`,
				Color:     `{{ template "slack.ridecell.color" . }}`,
				Text:      `{{ template "slack.ridecell.text" . }}`,
				Actions: []*alertmconfig.SlackAction{
					&alertmconfig.SlackAction{
						URL:  `{{ (index .Alerts 0).Annotations.runbook }}`,
						Text: "Runbook :green_book:",
						Type: "button",
					},
					&alertmconfig.SlackAction{
						URL:  fmt.Sprintf("https://%s/#/silences", os.Getenv("ALERTMANAGER_NAME")),
						Text: "Silence :no_bell:",
						Type: "button",
					},
					&alertmconfig.SlackAction{
						URL:  `{{ (index .Alerts 0).Annotations.dashboard }}`,
						Text: "Dashboard :grafana:",
						Type: "button",
					},
					&alertmconfig.SlackAction{
						URL:  `{{ (index .Alerts 0).GeneratorURL }}`,
						Text: "Query :mag:",
						Type: "button",
					},
				},
			})
		}
		extras["slack"] = receiverSlack
	}

	var eventid string
	if len(instance.Spec.Notify.PagerdutyTeam) > 0 {
		// Add add PD config in receiver
		receiverPD := &alertmconfig.Receiver{
			Name: instance.Name + "-pd",

			PagerdutyConfigs: []*alertmconfig.PagerdutyConfig{
				&alertmconfig.PagerdutyConfig{
					NotifierConfig: alertmconfig.NotifierConfig{
						VSendResolved: true,
					},
					RoutingKey:  alertmconfig.Secret(os.Getenv("PG_ROUTING_KEY")),
					Severity:    `{{ if .CommonLabels.severity }}{{ .CommonLabels.severity | toLower }}{{ else }}critical{{ end }}`,
					Client:      os.Getenv("ALERTMANAGER_NAME"),
					ClientURL:   fmt.Sprintf("https://%s", os.Getenv("ALERTMANAGER_NAME")),
					Description: `{{ template "pagerduty.default.description" .}}`},
			}}

		// Check if EscalationPolicy present in pagerduty
		ep := &pagerduty.EscalationPolicy{}
		lep, _, err := client.EscalationPolicies.List(&pagerduty.ListEscalationPoliciesOptions{
			Query: instance.Spec.Notify.PagerdutyTeam,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "Failed when checking EscalationPolicy %s", instance.Spec.Notify.PagerdutyTeam)
		}
		for _, e := range lep.EscalationPolicies {
			if e.Name == instance.Spec.Notify.PagerdutyTeam {
				ep = e
			}
		}
		if len(ep.Name) <= 0 {
			return components.Result{}, errors.Wrapf(err, "Not able to find EscalationPolicy %s", instance.Spec.Notify.PagerdutyTeam)
		}

		// Get service present or not
		lso := &pagerduty.ListServicesOptions{}
		lso.Query = instance.Spec.ServiceName
		lsr, _, err := client.Services.List(lso)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "Failed when checking service %s", instance.Spec.ServiceName)
		}

		// Check service present or not
		if len(lsr.Services) > 1 {
			return components.Result{}, errors.New("More than two service present with name: " + instance.Spec.ServiceName)
		} else if len(lsr.Services) == 0 {
			//create PD service
			service := pagerduty.Service{
				Name:        instance.Spec.ServiceName,
				Description: "This service created by ridecell-operator. Manual modification may break self service monitoring",
				IncidentUrgencyRule: &pagerduty.IncidentUrgencyRule{
					Type:    "constant",
					Urgency: "severity_based",
				},
				EscalationPolicy: &pagerduty.EscalationPolicyReference{
					HTMLURL: ep.HTMLURL,
					ID:      ep.ID,
					Self:    ep.Self,
					Summary: ep.Summary,
					Type:    ep.Type,
				},
				AlertCreation: "create_alerts_and_incidents",
			}
			s, _, err := client.Services.Create(&service)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "Failed to create service %s", instance.Spec.ServiceName)
			}

			// create global event rule.
			// This will will route events from alertmanager on th basics of ServiceName in PG
			var condition, conditions, action []interface{}

			condition = append(condition, "contains", []string{"path", "payload", "summary"}, instance.Spec.ServiceName)
			conditions = append(conditions, "or", condition)

			action = append(action, []string{"route", s.ID})

			event, _, err := client.EventRules.Create(&pagerduty.EventRule{
				Condition: conditions,
				CatchAll:  false,
				Actions:   action,
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "Failed to create event rule for service %s", instance.Spec.ServiceName)
			}
			eventid = event.ID
		} else if lsr.Services[0].EscalationPolicy.ID != ep.ID {
			lsr.Services[0].EscalationPolicy = &pagerduty.EscalationPolicyReference{
				HTMLURL: ep.HTMLURL,
				ID:      ep.ID,
				Self:    ep.Self,
				Summary: ep.Summary,
				Type:    ep.Type,
			}
			_, _, err := client.Services.Update(lsr.Services[0].ID, lsr.Services[0])
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update service %s with with escalation police %s",
					instance.Spec.ServiceName, ep.Name)
			}
		}
		extras["pd"] = receiverPD
	}

	_, _, err = ctx.CreateOrUpdate("alertmanagerconfig.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*monitoringv1beta1.AlertManagerConfig)
		existing := existingObj.(*monitoringv1beta1.AlertManagerConfig)
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to create AlertManagerConfig for %s", instance.Name)
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*monitoringv1beta1.Monitor)
		if len(eventid) > 0 {
			instance.Status.EventRuleID = eventid
		}
		return nil
	}}, nil

}
