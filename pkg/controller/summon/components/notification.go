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

	"github.com/nlopes/slack"
	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

var versionRegex *regexp.Regexp

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
	PostStatus(name string, env string, tag string) error
}

type realDeployStatusClient struct{}

// Real implementation of PostStatus for deployStatusTool
func (c *realDeployStatusClient) PostStatus(name string, env string, tag string) error {
	url := os.Getenv("DEPLOY_STAT_URL")

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

type notificationComponent struct {
	slackClient        SlackClient
	deployStatusClient DeployStatusClient
	dupCache           sync.Map
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
	}
}

func (c *notificationComponent) InjectSlackClient(client SlackClient) {
	c.slackClient = client
}

func (c *notificationComponent) InjectDeployStatusClient(client DeployStatusClient) {
	c.deployStatusClient = client
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

// Send a deploy notification if needed.
func (c *notificationComponent) handleSuccess(instance *summonv1beta1.SummonPlatform) (components.Result, error) {
	if instance.Spec.Version == instance.Status.Notification.NotifyVersion {
		// Already notified about this version, we're good.
		return components.Result{}, nil
	}

	// Check if this is a duplicate slipping through due to concurrency.
	dupCacheKey := fmt.Sprintf("%s/%s", instance.Namespace, instance.Name)
	lastdupCacheValue, ok := c.dupCache.Load(dupCacheKey)
	dupCacheValue := fmt.Sprintf("SUCCESS %s", instance.Spec.Version)
	if ok && lastdupCacheValue == dupCacheValue {
		return components.Result{}, nil
	}

	// Send to Slack.
	if instance.Spec.Notifications.SlackChannel != "" {
		attachment := c.formatSuccessNotification(instance)
		_, _, err := c.slackClient.PostMessage(instance.Spec.Notifications.SlackChannel, attachment)
		if err != nil {
			return components.Result{}, err
		}
	}

	// Send to Deployment Status Tool
	deployEnv := instance.Namespace
	if strings.HasPrefix(deployEnv, "summon-") {
		deployEnv = deployEnv[7:]
	}

	err := c.deployStatusClient.PostStatus(instance.Name, deployEnv, instance.Spec.Version)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "notifications: error posting to deployment-status")
	}

	// Update status. Close over `version` in case it changes during a collision.
	c.dupCache.Store(dupCacheKey, dupCacheValue)
	version := instance.Spec.Version
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*summonv1beta1.SummonPlatform)
		instance.Status.Notification.NotifyVersion = version
		return nil
	}}, nil
}

// Send an error notification if needed.
func (c *notificationComponent) handleError(instance *summonv1beta1.SummonPlatform, errorMessage string) (components.Result, error) {
	// Check if this is a duplicate message.
	dupCacheKey := fmt.Sprintf("%s/%s", instance.Namespace, instance.Name)
	lastdupCacheValue, ok := c.dupCache.Load(dupCacheKey)
	dupCacheValue := fmt.Sprintf("ERROR %s", errorMessage)
	if ok && lastdupCacheValue == dupCacheValue {
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

	// Update status.
	c.dupCache.Store(dupCacheKey, dupCacheValue)
	return components.Result{}, nil
}

// Render the nofiication attachement for a deploy notification.
func (comp *notificationComponent) formatSuccessNotification(instance *summonv1beta1.SummonPlatform) slack.Attachment {
	fields := []slack.AttachmentField{}
	// Try to parse the version string using our usual conventions.
	matches := versionRegex.FindStringSubmatch(instance.Spec.Version)
	if matches != nil {
		// Build fields for each thing.
		buildField := slack.AttachmentField{
			Title: "Build",
			Value: fmt.Sprintf("<https://circleci.com/gh/Ridecell/summon-platform/%s|%s>", matches[1], matches[1]),
			Short: true,
		}
		shaField := slack.AttachmentField{
			Title: "Commit",
			Value: fmt.Sprintf("<https://github.com/Ridecell/summon-platform/tree/%s|%s>", matches[2], matches[2]),
			Short: true,
		}
		branchField := slack.AttachmentField{
			Title: "Branch",
			Value: fmt.Sprintf("<https://github.com/Ridecell/summon-platform/tree/%s|%s>", matches[3], matches[3]),
			Short: true,
		}
		fields = append(fields, shaField, branchField, buildField)
	}

	return slack.Attachment{
		Title:     fmt.Sprintf("%s Deployment", instance.Spec.Hostname),
		TitleLink: fmt.Sprintf("https://%s/", instance.Spec.Hostname),
		Color:     "good",
		Text:      fmt.Sprintf("<https://%s/|%s> deployed version %s successfully", instance.Spec.Hostname, instance.Spec.Hostname, instance.Spec.Version),
		Fallback:  fmt.Sprintf("%s deployed version %s successfully", instance.Spec.Hostname, instance.Spec.Version),
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
