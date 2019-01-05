/*
Copyright 2018 Ridecell, Inc.

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

package v1beta1

import (
	"time"

	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Gross workaround for limitations the Kubernetes code generator and interface{}.
// If you want to see the weird inner workings of the hack, looking marshall.go.
type ConfigValue struct {
	Bool   *bool    `json:"bool,omitempty"`
	Float  *float64 `json:"float,omitempty"`
	String *string  `json:"string,omitempty"`
}

// NotificationSecretRef defines the spec for the slack API secret
type NotificationSecretRef struct {
	Name string `json:"name"`
	Key  string `json:"key,omitempty"`
}

// NotificationStatus defines the observed state of Notifications
type NotificationStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	// +optional
	NotifyVersion string `json:"notifyVersion,omitempty"`
	LastErrorHash string `json:"lastErrorHash,omitempty"`
}

// SummonPlatformSpec defines the desired state of SummonPlatform
type SummonPlatformSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Hostname to use for the instance. Defaults to $NAME.ridecell.us.
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Summon image version to deploy.
	Version string `json:"version"`
	// Name of the secret to use for secret values.
	Secret string `json:"secret"`
	// Name of the secret to use for image pulls. Defaults to `"pull-secret"`.
	// +optional
	PullSecret string `json:"pullSecret,omitempty"`
	// Summon-platform.yml configuration options.
	Config map[string]ConfigValue `json:"config,omitempty"`
	// Number of gunicorn pods to run. Defaults to 1.
	// +optional
	WebReplicas *int32 `json:"web_replicas,omitempty"`
	// Number of daphne pods to run. Defaults to 1.
	// +optional
	DaphneReplicas *int32 `json:"daphne_replicas,omitempty"`
	// Number of celeryd pods to run. Defaults to 1.
	// +optional
	WorkerReplicas *int32 `json:"worker_replicas,omitempty"`
	// Number of channelworker pods to run. Defaults to 1.
	// +optional
	ChannelWorkerReplicas *int32 `json:"channel_worker_replicas,omitempty"`
	// Number of caddy pods to run. Defaults to 1.
	// +optional
	StaticReplicas *int32 `json:"static_replicas,omitempty"`
	// Slack API endpoint
	// +optional
	SlackAPIEndpoint string `json:"slackApiEndpoint,omitempty"`
	// Name of the slack channel for notifications. Defaults to "".
	// +optional
	SlackChannelName string `json:"slackChannelName,omitempty"`
	// Slack API Key Secret Definition
	// +optional
	NotificationSecretRef NotificationSecretRef `json:"secretRef,omitempty"`
	// Fernet Key Rotation Time Setting
	// +optional
	FernetKeyRotation time.Duration `json:"fernetKeyRotation,omitempty"`
}

// SummonPlatformStatus defines the observed state of SummonPlatform
type SummonPlatformStatus struct {
	// Overall object status
	Status string `json:"status,omitempty"`

	// Message related to the current status.
	Message string `json:"message,omitempty"`

	// Status of the pull secret.
	PullSecretStatus string `json:"pullSecretStatus,omitempty"`

	// Current Postgresql status if one exists.
	PostgresStatus postgresv1.PostgresStatus `json:"postgresStatus,omitempty"`

	// Status of the required Postgres extensions (collectively).
	PostgresExtensionStatus string `json:"postgresExtensionStatus,omitempty"`

	// Previous version for which migrations ran successfully.
	// +optional
	MigrateVersion string `json:"migrateVersion,omitempty"`
	// Spec for Notification
	// +optional
	Notification NotificationStatus `json:"notification,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SummonPlatform is the Schema for the summonplatforms API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SummonPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SummonPlatformSpec   `json:"spec,omitempty"`
	Status SummonPlatformStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SummonPlatformList contains a list of SummonPlatform
type SummonPlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SummonPlatform `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SummonPlatform{}, &SummonPlatformList{})
}
