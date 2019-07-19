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

package v1beta1

import (
	"time"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Gross workaround for limitations the Kubernetes code generator and interface{}.
// If you want to see the weird inner workings of the hack, look in marshall.go.
type ConfigValue struct {
	Bool   *bool    `json:"bool,omitempty"`
	Float  *float64 `json:"float,omitempty"`
	String *string  `json:"string,omitempty"`
}

// NotificationsSpec defines notificiations settings for this instance.
type NotificationsSpec struct {
	// Name of the slack channel for notifications. If not set, no notifications will be sent.
	// +optional
	SlackChannel string `json:"slackChannel,omitempty"`
	// list of additional slack channels for notifications
	// +optional
	SlackChannels []string `json:"slackChannels,omitempty"`
	// Override for the global default deployment-status server to use.
	// +optional
	DeploymentStatusUrl string `json:"deploymentStatusUrl,omitempty"`
}

// DatabaseSpec is used to specify whether we are using a shared database or not.
type DatabaseSpec struct {
	// +optional
	ExclusiveDatabase bool `json:"exclusiveDatabase,omitempty"`
	// +optional
	SharedDatabaseName string `json:"sharedDatabaseName,omitempty"`
}

// MIVSpec defines the configuration of the Manual Identiy Verification bucket feature.
type MIVSpec struct {
	// The optional name of an existing S3 bucket to use. If set, this code does not create its own bucket.
	// +optional
	ExistingBucket string `json:"existingBucket,omitempty"`
}

// BackupSpec defines the configuration of the automatic RDS Snapshot feature.
type BackupSpec struct {
	// The ttl of the created rds snapshot in string form.
	// +optional
	TTL metav1.Duration `json:"ttl,omitempty"`
	// whether or not the backup process waits on the snapshot to finish
	WaitUntilReady *bool `json:"waitUntilReady,omitempty"`
}

// WaitSpec defines the configuration of post migration delays.
type WaitSpec struct {
	PostMigrate metav1.Duration `json:"postMigrate,omitempty"`
}

// SummonPlatformSpec defines the desired state of SummonPlatform
type SummonPlatformSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Hostname to use for the instance. Defaults to $NAME.ridecell.us.
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Hostname aliases (for vanity purposes)
	// +optional
	Aliases []string `json:"aliases,omitempty"`
	// Summon image version to deploy.
	Version string `json:"version"`
	// Name of the secret to use for secret values.
	Secrets []string `json:"secrets,omitempty"`
	// Name of the secret to use for image pulls. Defaults to `"pull-secret"`.
	// +optional
	PullSecret string `json:"pullSecret,omitempty"`
	// Summon-platform.yml configuration options.
	Config map[string]ConfigValue `json:"config,omitempty"`
	// Number of gunicorn pods to run. Defaults to 1.
	// +optional
	WebReplicas *int32 `json:"webReplicas,omitempty"`
	// Number of daphne pods to run. Defaults to 1.
	// +optional
	DaphneReplicas *int32 `json:"daphneReplicas,omitempty"`
	// Number of celeryd pods to run. Defaults to 1.
	// +optional
	WorkerReplicas *int32 `json:"workerReplicas,omitempty"`
	// Number of channelworker pods to run. Defaults to 1.
	// +optional
	ChannelWorkerReplicas *int32 `json:"channelWorkerReplicas,omitempty"`
	// Number of caddy pods to run. Defaults to 1.
	// +optional
	StaticReplicas *int32 `json:"staticReplicas,omitempty"`
	// Settings for deploy and error notifications.
	// +optional
	Notifications NotificationsSpec `json:"notifications,omitempty"`
	// Fernet Key Rotation Time Setting
	// +optional
	FernetKeyLifetime time.Duration `json:"fernetKeyLifetime,omitempty"`
	// Disable the creation of the dispatcher@ridecell.com superuser.
	NoCreateSuperuser bool `json:"noCreateSuperuser,omitempty"`
	// AWS Region setting
	// +optional
	AwsRegion string `json:"awsRegion,omitempty"`
	// SQS queue setting
	// +optional
	SQSQueue string `json:"sqsQueue,omitempty"`
	// SQS queue region setting
	// +optional
	SQSRegion string `json:"sqsRegion,omitempty"`
	// Database-related settings.
	// +optional
	Database DatabaseSpec `json:"database,omitempty"`
	// The flavor of data to be imported upon creation
	// +optional
	Flavor string `json:"flavor,omitempty"`
	// Manual Identity Verification settings.
	// +optional
	MIV MIVSpec `json:"miv,omitempty"`
	// Environment setting.
	// +optional
	Environment string `json:"environment,omitempty"`
	// Enable NewRelic APM.
	// +optional
	EnableNewRelic *bool `json:"enableNewRelic,omitempty"`
	// Automated backup settings.
	// +optional
	Backup BackupSpec `json:"backup,omitempty"`
	// Deployment wait settings
	// +optional
	Waits WaitSpec `json:"waits,omitempty"`
}

// NotificationStatus defines the observed state of Notifications
type NotificationStatus struct {
	// The last version we posted a deploy success notification for.
	// +optional
	NotifyVersion string `json:"notifyVersion,omitempty"`
}

// MIVStatus is the output information for the Manual Identity Verification system.
type MIVStatus struct {
	// The MIV data S3 bucket name.
	Bucket string `json:"bucket,omitempty"`
}

// WaitStatus is the output information for deployment Waits.
type WaitStatus struct {
	// The time that deployments should wait for after migrations to continue.
	// Real type = time.Time
	// workaround because metav1.Time is broked
	// +optional
	Until string `json:"until,omitempty"`
}

// SummonPlatformStatus defines the observed state of SummonPlatform
type SummonPlatformStatus struct {
	// Overall object status
	Status string `json:"status,omitempty"`

	// Message related to the current status.
	Message string `json:"message,omitempty"`

	// Status of the pull secret.
	PullSecretStatus string `json:"pullSecretStatus,omitempty"`

	// Current PostgresDatabase status if one exists.
	PostgresStatus     string                       `json:"postgresStatus,omitempty"`
	PostgresConnection dbv1beta1.PostgresConnection `json:"postgresConnection,omitempty"`

	// Current RabbitMQ status if one exists.
	RabbitMQStatus     string                             `json:"rabbitmqStatus,omitempty"`
	RabbitMQConnection dbv1beta1.RabbitmqStatusConnection `json:"rabbitmqConnection,omitempty"`

	// Previous version for which migrations ran successfully.
	// +optional
	MigrateVersion string `json:"migrateVersion,omitempty"`
	// Previous version for which a backup was made.
	// +optional
	BackupVersion string `json:"backupVersion,omitempty"`
	// Spec for Notification
	// +optional
	Notification NotificationStatus `json:"notification,omitempty"`
	// Status for MIV system.
	// +optional
	MIV MIVStatus `json:"miv,omitempty"`
	// Status for deployment Waits
	// +optional
	Wait WaitStatus `json:"wait,omitempty"`
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
