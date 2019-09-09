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
	"os"

	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/utils/sumologic"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

const logruleFinalizer = "finalizer.logrule.monitoring.ridecell.io"

// AlertFolderid ID of alert folder in sumologic
const AlertFolderid = "000000000083D019"

type logruleComponent struct {
}

func NewLogrule() *logruleComponent {
	return &logruleComponent{}
}

func (_ *logruleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&monitoringv1beta1.Monitor{},
	}
}

func (_ *logruleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *logruleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*monitoringv1beta1.Monitor)

	// absence MetricAlertRules should not retrun error else other components will break
	if len(instance.Spec.LogAlertRules) <= 0 {
		return components.Result{}, nil
	}
	// Create sumologic client
	client, _ := sumologic.NewClient("https://api.us2.sumologic.com", os.Getenv("SUMO_ACCESS_ID"), os.Getenv("SUMO_ACCESS_KEY"))
	// Run with mockserver
	if len(os.Getenv("SUMO_MOCK_URL")) > 0 {
		client, _ = sumologic.NewClient(os.Getenv("SUMO_MOCK_URL"), os.Getenv("SUMO_ACCESS_ID"), os.Getenv("SUMO_ACCESS_KEY"))
	}
	connections, err := client.ListConnections()
	if err != nil {
		return components.Result{}, errors.Wrap(err, "Failed to list connections")
	}

	// Check connection
	connectionToUse := ""
	for _, connection := range connections.Data {
		if connection.Name == os.Getenv("ALERTMANAGER_NAME") {
			connectionToUse = connection.ID
		}

	}
	// Create connection
	connection := sumologic.WebHookConnection{
		Type:           "WebhookDefinition",
		Name:           os.Getenv("ALERTMANAGER_NAME"),
		Description:    "Created by ridecell-operator do NOT modify",
		URL:            fmt.Sprintf("https://%s/api/v1/alerts", os.Getenv("ALERTMANAGER_NAME")),
		DefaultPayload: "{}",
		Headers: []sumologic.WebHookHeaders{
			{Name: "Authorization", Value: "Basic " + base64.StdEncoding.EncodeToString([]byte(os.Getenv("ALERTMANAGER_AUTH")))},
		},
		WebhookType: "Webhook",
	}
	if len(connectionToUse) <= 0 {
		_, err = client.CreateConnection(connection)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "failed to create new connection")
		}
		connectionToUse = connection.ID
	}

	//Find service folder
	var serviceFolderid string
	folders, err := client.GetFolder(AlertFolderid)
	for _, folder := range folders.Children {
		if instance.Spec.ServiceName == folder.Name {
			serviceFolderid = folder.ID
		}
	}
	if len(serviceFolderid) <= 0 {
		resp, err := client.CreateFolder(sumologic.Folder{
			Name:        instance.Spec.ServiceName,
			Description: fmt.Sprintf("Folder is created by ridecell-operator to store alerts for service %s", instance.Spec.ServiceName),
			ParentID:    AlertFolderid,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "Failed to create alert folder for service %s", instance.Spec.ServiceName)
		}
		serviceFolderid = resp.ID
	}

	// Finalizer start here
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(logruleFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(logruleFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(logruleFinalizer, instance) {
			contents, err := client.GetFolder(serviceFolderid)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "Failed to get folder at the time Finalizer")
			}
			for _, content := range contents.Children {
				for _, rule := range instance.Spec.LogAlertRules {
					if rule.Name == content.Name && content.ItemType == "Search" {
						_, err := client.DeleteContent(content.ID)
						if err != nil {
							return components.Result{}, errors.Wrapf(err, "failed to delete rule with name %s", rule.Name)
						}

					}

				}
			}

			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(logruleFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update LogAlertsRule while removing finalizer")
			}
		}
		return components.Result{}, nil
	}

	for _, rule := range instance.Spec.LogAlertRules {
		//
		scheduleType := "Custom"

		// Create payload for every search
		payload := fmt.Sprintf(`[{ 
			"status": "firing",
			"labels": {
					"alertname": "{{SearchName}}",
					"service": "%s",
					"severity":"%s"
			},
			"annotations": {
					"summary": "{{SearchDescription}}",
					"runbook": "%s"
			}
	}]`, instance.Spec.ServiceName, rule.Severity, rule.Runbook)
		search := sumologic.SavedSearchWithSchedule{

			Type:        "SavedSearchWithScheduleSyncDefinition",
			Name:        rule.Name,
			Description: rule.Description,
			Search: sumologic.Search{
				QueryText:        rule.Query,
				DefaultTimeRange: "-5m",
				ByReceiptTime:    false,
				QueryParameters:  []sumologic.QueryParameters{},
			},
			SearchSchedule: sumologic.SearchSchedule{
				DisplayableTimeRange: rule.Range,
				ParseableTimeRange: sumologic.ParseableTimeRange{
					Type: "BeginBoundedTimeRange",
					From: sumologic.From{
						RelativeTime: rule.Range,
						Type:         "RelativeTimeRangeBoundary",
					},
					To: nil,
				},

				TimeZone: "Etc/UTC",
				Threshold: sumologic.Threshold{
					ThresholdType: "group",
					Operator:      rule.Condition,
					Count:         rule.Threshold,
				},
				Notification: sumologic.Notification{
					TaskType:  "WebhookSearchNotificationSyncDefinition",
					WebhookId: connectionToUse,
					Payload:   payload,
				},
				ScheduleType: scheduleType,
				Parameters:   []sumologic.Parameters{},
			},
			AutoParsingData: sumologic.AutoParsingData{
				Mode: "performance",
			},
		}

		if rule.Schedule == "RealTime" {
			search.SearchSchedule.ScheduleType = "RealTime"
		} else {
			search.SearchSchedule.ScheduleType = "Custom"
			search.SearchSchedule.CronExpression = rule.Schedule
		}

		_, err := client.CreateSavedSearchWithSchedule(serviceFolderid, &search, true)
		if err != nil {
			return components.Result{}, errors.Wrapf(err,
				`Failed to create search with name "%s" for "%s" in namespace %s`,
				rule.Name, instance.Name, instance.Namespace)
		}
	}
	return components.Result{}, nil
}
