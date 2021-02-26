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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/slack-go/slack"
	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

var versionRegex *regexp.Regexp

// Types to identify which component we handle notifications for. These map to their github repo names.
const CompSummonStr = "summon-platform"
const CompDispatchStr = "comp-dispatch"
const CompBusinessPortalStr = "comp-business-portal"
const CompHwAuxStr = "comp-hw-aux"
const CompTripShareStr = "comp-trip-share"
const CompPulseStr = "comp-pulse"
const CompCustomerPortal = "comp-customer-portal"

func init() {
	versionRegex = regexp.MustCompile(`^(\d+)-([0-9a-fA-F]+)-(\S+)$`)
}

// Interface for a Slack client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_slackclient_test.go . SlackClient
type SlackClient interface {
	PostMessage(string, slack.Attachment) (string, string, error)
}

// Real implementation of SlackClient using nlopes/slack.
// I can't match the interface to that directly because the MsgOptions API involves
// private structs so I can't actually get the back out the other side when working with a mock.
type realSlackClient struct {
	client *slack.Client
}

func (c *realSlackClient) PostMessage(channel string, msg slack.Attachment) (string, string, error) {
	if c.client != nil {
		return c.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
	} else {
		return "", "", nil
	}
}

// Interface for Deployment status client
//go:generate moq -out zz_generated.mock_deploystatusclient_test.go . DeployStatusClient
type DeployStatusClient interface {
	PostStatus(url string, name string, env string, tag string) error
}

type realDeployStatusClient struct{}

// Real implementation of PostStatus for deployStatusTool
func (c *realDeployStatusClient) PostStatus(url string, name string, env string, tag string) error {
	if url == "" {
		return nil
	}

	postBody := map[string]string{
		"customer_name": name,
		"environment":   env,
		"deploy_user":   "ridecell-operator", // better candidate user?
		"tag":           tag,
	}

	postJson, err := json.Marshal(postBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(postJson))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyContent, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.Errorf("error from deployment status %v: %s", resp.StatusCode, bodyContent)
	}
	return nil
}

// Interface for Circleci client
//go:generate moq -out zz_generated.mock_circleciclient_test.go . CircleCiClient
type CircleCiClient interface {
	TriggerRegressionSuite(instanceName string, version string) string
}

type realCircleCiClient struct{}

// Real implementation of CallCircleciAPI
func (c *realCircleCiClient) TriggerRegressionSuite(instanceName string, version string) string {
	apiKey := os.Getenv("CIRCLECI_API_KEY")
	if apiKey != "" {
		postData := map[string]interface{}{
			"branch": "master",
			"parameters": map[string]interface{}{
				"deployment-regression-tests": true,
				"framework-tests":             false,
				"tenant-name":                 instanceName,
				"build-tag":                   version,
			},
		}
		err := utils.CallCircleCiWebhook("https://circleci.com/api/v2/project/github/Ridecell/Ridecell_qa_automation/pipeline", apiKey, postData)
		if err != nil {
			return fmt.Sprintf("%s", err)
		}
	} else {
		return "CIRCLECI_API_KEY environment variable not defined"
	}
	return "Success"
}

type notificationComponent struct {
	slackClient        SlackClient
	deployStatusClient DeployStatusClient
	dupCache           sync.Map
	circleciClient     CircleCiClient
}

func NewNotification() *notificationComponent {
	var slackClient *slack.Client
	slackApiKey := os.Getenv("SLACK_API_KEY")
	if slackApiKey != "" {
		slackClient = slack.New(slackApiKey)
	}

	return &notificationComponent{
		slackClient:        &realSlackClient{client: slackClient},
		deployStatusClient: &realDeployStatusClient{},
		circleciClient:     &realCircleCiClient{},
	}
}

func (c *notificationComponent) InjectSlackClient(client SlackClient) {
	c.slackClient = client
}

func (c *notificationComponent) InjectDeployStatusClient(client DeployStatusClient) {
	c.deployStatusClient = client
}

func (c *notificationComponent) InjectCircleCiClient(client CircleCiClient) {
	c.circleciClient = client
}

func (_ *notificationComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *notificationComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (c *notificationComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	if instance.Status.Status == summonv1beta1.StatusReady {
		return c.handleSuccess(instance)
	} else if instance.Status.Status == summonv1beta1.StatusError {
		return c.handleError(instance, instance.Status.Message)
	}

	// No notifications needed.
	return components.Result{}, nil
}

// ReconcileError implements components.ErrorHandler.
func (c *notificationComponent) ReconcileError(ctx *components.ComponentContext, err error) (components.Result, error) {
	if !errors.ShouldNotify(err) {
		return components.Result{}, nil
	}
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	return c.handleError(instance, fmt.Sprintf("%s", err))
}

// Checks each summon component and send a deploy notification if needed.
func (c *notificationComponent) handleSuccess(instance *summonv1beta1.SummonPlatform) (components.Result, error) {

	// Accumulate errors to be dealt with at the end so no component notifications
	// are blocked on another's error.
	var errs error
	if instance.Spec.Version != instance.Status.Notification.SummonVersion {
		err := c.notifyAndPostStatus(instance, CompSummonStr, instance.Spec.Version)
		if err != nil {
			errs = err
		}
	}
	if instance.Spec.Dispatch.Version != instance.Status.Notification.DispatchVersion {
		err := c.notifyAndPostStatus(instance, CompDispatchStr, instance.Spec.Dispatch.Version)
		if err != nil {
			errs = fmt.Errorf("%s; %s", errs, err)
		}
	}
	if instance.Spec.BusinessPortal.Version != instance.Status.Notification.BusinessPortalVersion {
		err := c.notifyAndPostStatus(instance, CompBusinessPortalStr, instance.Spec.BusinessPortal.Version)
		if err != nil {
			errs = fmt.Errorf("%s; %s", errs, err)
		}
	}
	if instance.Spec.Pulse.Version != instance.Status.Notification.PulseVersion {
		err := c.notifyAndPostStatus(instance, CompPulseStr, instance.Spec.Pulse.Version)
		if err != nil {
			errs = fmt.Errorf("%s; %s", errs, err)
		}
	}
	if instance.Spec.CustomerPortal.Version != instance.Status.Notification.CustomerPortalVersion {
		err := c.notifyAndPostStatus(instance, CompCustomerPortal, instance.Spec.CustomerPortal.Version)
		if err != nil {
			errs = fmt.Errorf("%s; %s", errs, err)
		}
	}
	if instance.Spec.HwAux.Version != instance.Status.Notification.HwAuxVersion {
		err := c.notifyAndPostStatus(instance, CompHwAuxStr, instance.Spec.HwAux.Version)
		if err != nil {
			errs = fmt.Errorf("%s; %s", errs, err)
		}
	}
	if instance.Spec.TripShare.Version != instance.Status.Notification.TripShareVersion {
		err := c.notifyAndPostStatus(instance, CompTripShareStr, instance.Spec.TripShare.Version)
		if err != nil {
			errs = fmt.Errorf("%s; %s", errs, err)
		}
	}
	if errs != nil {
		return components.Result{}, errs
	}

	// Trigger CircleCi Regression Test Webhook, if set true
	var webhookStatus string
	if instance.Spec.Notifications.CircleciRegressionWebhook {
		// Check if this is a duplicate slipping through due to concurrency.
		dupCacheKey := fmt.Sprintf("%s/%s", instance.Namespace, instance.Name)
		lastdupCacheValue, ok := c.dupCache.Load(dupCacheKey)
		if !ok || lastdupCacheValue != instance.Spec.Version {
			webhookStatus = c.circleciClient.TriggerRegressionSuite(instance.Name, instance.Spec.Version)
			if webhookStatus == "Success" {
				c.dupCache.Store(dupCacheKey, instance.Spec.Version)
			}
		}
	}

	// no errors, update notification statuses.
	return components.Result{
		StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Notification.SummonVersion = instance.Spec.Version
			instance.Status.Notification.DispatchVersion = instance.Spec.Dispatch.Version
			instance.Status.Notification.BusinessPortalVersion = instance.Spec.BusinessPortal.Version
			instance.Status.Notification.PulseVersion = instance.Spec.Pulse.Version
			instance.Status.Notification.HwAuxVersion = instance.Spec.HwAux.Version
			instance.Status.Notification.TripShareVersion = instance.Spec.TripShare.Version
			instance.Status.Notification.CustomerPortalVersion = instance.Spec.CustomerPortal.Version
			if instance.Spec.Notifications.CircleciRegressionWebhook {
				instance.Status.Notification.CircleciRegressionWebhook = webhookStatus
			}
			return nil
		}}, nil
}

func (c *notificationComponent) notifyAndPostStatus(instance *summonv1beta1.SummonPlatform, component string, version string) error {

	// Check if this is a duplicate slipping through due to concurrency.
	dupCacheKey := fmt.Sprintf("%s/%s-%s", instance.Namespace, instance.Name, component)
	lastdupCacheValue, ok := c.dupCache.Load(dupCacheKey)
	dupCacheValue := fmt.Sprintf("SUCCESS %s", version)
	if ok && lastdupCacheValue == dupCacheValue {
		return nil
	}

	// Send to Slack.
	if instance.Spec.Notifications.SlackChannel != "" {
		attachment := c.formatSuccessNotification(instance, component, version)
		_, _, err := c.slackClient.PostMessage(instance.Spec.Notifications.SlackChannel, attachment)
		if err != nil {
			return err
		}
	}

	// Send to additional slack channels.
	for _, channel := range instance.Spec.Notifications.SlackChannels {
		attachment := c.formatSuccessNotification(instance, component, version)
		_, _, err := c.slackClient.PostMessage(channel, attachment)
		if err != nil {
			return err
		}
	}

	// Send to Deployment Status Tool
	instanceName := strings.TrimSuffix(instance.Name, "-"+instance.Spec.Environment)
	if component != CompSummonStr {
		// Modify format if we're dealing with summon component (not the platform).
		instanceName = instanceName + " " + component
	}

	deploymentStatusUrl := os.Getenv("DEPLOY_STAT_URL")
	if instance.Spec.Notifications.DeploymentStatusUrl != "" {
		deploymentStatusUrl = instance.Spec.Notifications.DeploymentStatusUrl
	}
	err := c.deployStatusClient.PostStatus(deploymentStatusUrl, instanceName, instance.Spec.Environment, version)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("notifications: error posting to deployment-status for %s", instanceName))
	}
	c.dupCache.Store(dupCacheKey, dupCacheValue)
	return nil
}

// Send an error notification if needed.
func (c *notificationComponent) handleError(instance *summonv1beta1.SummonPlatform, errorMessage string) (components.Result, error) {
	// Check if this is a duplicate message.
	dupCacheKey := fmt.Sprintf("%s/%s/%s", instance.Namespace, instance.Name, instance.Spec.Version)
	lastdupCacheValue, ok := c.dupCache.Load(dupCacheKey)
	dupCacheValue := fmt.Sprintf("ERROR %s", errorMessage)
	if ok && lastdupCacheValue == dupCacheValue {
		return components.Result{}, nil
	}

	// Check if this is one of the "the object has been modified; please apply your changes to the latest version and try again" errors.
	if strings.Contains(errorMessage, "the object has been modified; please apply your changes to the latest version and try again") {
		// TODO warning log should go here.
		return components.Result{}, nil
	}

	// Send to Slack.
	if instance.Spec.Notifications.SlackChannel != "" {
		attachment := c.formatErrorNotification(instance, errorMessage)
		_, _, err := c.slackClient.PostMessage(instance.Spec.Notifications.SlackChannel, attachment)
		if err != nil {
			return components.Result{}, err
		}
	}

	// Send to additonal slack channels
	for _, channel := range instance.Spec.Notifications.SlackChannels {
		attachment := c.formatErrorNotification(instance, errorMessage)
		_, _, err := c.slackClient.PostMessage(channel, attachment)
		if err != nil {
			return components.Result{}, err
		}
	}

	// Update status.
	c.dupCache.Store(dupCacheKey, dupCacheValue)
	return components.Result{}, nil
}

// Render the notification attachement for a deploy notification.
func (comp *notificationComponent) formatSuccessNotification(instance *summonv1beta1.SummonPlatform, component string, version string) slack.Attachment {
	fields := []slack.AttachmentField{}
	// Try to parse the version string using our usual conventions.
	matches := versionRegex.FindStringSubmatch(version)
	if matches != nil {
		// Build fields for each thing.
		buildField := slack.AttachmentField{
			Title: "Build",
			Value: fmt.Sprintf("<https://circleci.com/gh/Ridecell/%s/%s|%s>", component, matches[1], matches[1]),
			Short: true,
		}
		shaField := slack.AttachmentField{
			Title: "Commit",
			Value: fmt.Sprintf("<https://github.com/Ridecell/%s/tree/%s|%s>", component, matches[2], matches[2]),
			Short: true,
		}
		branchField := slack.AttachmentField{
			Title: "Branch",
			Value: fmt.Sprintf("<https://github.com/Ridecell/%s/tree/%s|%s>", component, matches[3], matches[3]),
			Short: true,
		}
		fields = append(fields, shaField, branchField, buildField)
	}

	return slack.Attachment{
		Title:     fmt.Sprintf("%s %s Deployment", instance.Spec.Hostname, component),
		TitleLink: fmt.Sprintf("https://%s/", instance.Spec.Hostname),
		Color:     "good",
		Text:      fmt.Sprintf("<https://%s/|%s> deployed %s version %s successfully", instance.Spec.Hostname, instance.Spec.Hostname, component, version),
		Fallback:  fmt.Sprintf("%s deployed %s version %s successfully", instance.Spec.Hostname, component, version),
		Fields:    fields,
	}
}

// Render the nofiication attachement for an error notification.
func (comp *notificationComponent) formatErrorNotification(instance *summonv1beta1.SummonPlatform, errorMessage string) slack.Attachment {
	return slack.Attachment{
		Title:     fmt.Sprintf("%s Deployment", instance.Spec.Hostname),
		TitleLink: fmt.Sprintf("https://%s/", instance.Spec.Hostname),
		Color:     "danger",
		Text:      fmt.Sprintf("<https://%s/|%s> has error: %s", instance.Spec.Hostname, instance.Spec.Hostname, errorMessage),
		Fallback:  fmt.Sprintf("%s has error: %s", instance.Spec.Hostname, errorMessage),
	}
}
