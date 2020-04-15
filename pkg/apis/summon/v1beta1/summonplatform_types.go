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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
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
	// Name of pagerduty team. Team will be paged for all critical alerts
	Pagerdutyteam string `json:"pagerdutyteam,omitempty"`
}

// DatabaseSpec defines database-related configuration.
type DatabaseSpec struct {
	// An optional ref to a DbConfig object to use for configuration. Defaults to the name of the namespace.
	// +optional
	DbConfigRef corev1.ObjectReference `json:"dbConfigRef,omitempty"`
}

// CelerySpec defines configuration and settings for Celery.
type CelerySpec struct {
	// Setting for --concurrency.
	// +optional
	Concurrency int `json:"concurrency,omitempty"`
	// Setting for --pool.
	// +optional
	// +kubebuilder:validation:Enum=prefork,eventlet,gevent,solo
	Pool string `json:"pool,omitempty"`
}

// RedisSpec defines resource configuration for redis deployment.
type RedisSpec struct {
	// Setting for tuning redis memory request/limit in MB (or GB if over 10).
	// +optional
	RAM int `json:"ram,omitempty"`
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

// MigrationOverridesSpec defines value overrides used when migrating Ansible-based Summon instances into Kubernetes/ridecell-operator.
type MigrationOverridesSpec struct {
	RDSInstanceID     string `json:"rdsInstanceId,omitempty"`
	RDSMasterUsername string `json:"rdsMasterUsername,omitempty"`
	PostgresDatabase  string `json:"postgresDatabase,omitempty"`
	PostgresUsername  string `json:"postgresUsername,omitempty"`
	RabbitMQVhost     string `json:"rabbitmqVhost,omitempty"`
	RedisHostname     string `json:"redisHostname,omitempty"`
}

// ReplicasSpec defines the number of replicas of various types of pods to run.
type ReplicasSpec struct {
	// Number of web (twisted) pods to run. Defaults to 1 for dev/qa, 2 for uat, 4 for prod.
	// +optional
	Web *int32 `json:"web,omitempty"`
	// Number of daphne pods to run. Defaults to 1 for dev/qa, 2 for uat/prod.
	// +optional
	Daphne *int32 `json:"daphne,omitempty"`
	// Number of celeryd pods to run. Defaults to 1 for dev/qa/uat, 4 for prod.
	// +optional
	Celeryd *int32 `json:"celeryd,omitempty"`
	// Use horizontal pod autoscaling instead of a set replica.
	// +optional
	CelerydAuto bool `json:"celerydAuto,omitempty"`
	// Number of celerybeat pods to run. Defaults to 1. Must be exactly 0 or 1.
	// +optional
	CeleryBeat *int32 `json:"celeryBeat,omitempty"`
	// Number of channelworker pods to run. Defaults to 1 for dev/qa, 2 for uat, 4 for prod.
	// +optional
	ChannelWorker *int32 `json:"channelWorker,omitempty"`
	// Number of caddy pods to run. Defaults to 1 for dev/qa, 2 for uat/prod.
	// +optional
	Static *int32 `json:"static,omitempty"`
	// Number of dispatch pods to run. Defaults to 1 for dev/qa, 2 for uat/prod. Overridden to 0 if no dispatch.version is set.
	// +optional
	Dispatch *int32 `json:"dispatch,omitempty"`
	// Number of business-portal pods to run. Defaults to 1 for dev/qa, 2 for uat/prod. Overridden to 0 if no businessPortal.version is set.
	// +optional
	BusinessPortal *int32 `json:"businessPortal,omitempty"`
	// Number of trip-share pods to run. Defaults to 1 for dev/qa, 2 for uat/prod. Overridden to 0 if no tripShare.version is set.
	// +optional
	TripShare *int32 `json:"tripShare,omitempty"`
	// Number of hw-aux pods to run. Defaults to 1 for dev/qa, 2 for uat/prod. Overridden to 0 if no hwAux.version is set.
	// +optional
	HwAux *int32 `json:"hwAux,omitempty"`
}

// MonitorSpec will enable in monitoring. (In future we can use it to configure monitor.ridecell.io)
type MonitoringSpec struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// MetricsSpec defines what metrics should be enabled and exported
type MetricsSpec struct {
	// Enables metrics exporting for web pods
	Web *bool `json:"web,omitempty"`
}

// CompDispatchSpec defines settings for comp-dispatch.
type CompDispatchSpec struct {
	// Comp-dispatch image version to deploy.
	Version string `json:"version"`
}

// CompBusinessPortalSpec defines settings for comp-business-portal.
type CompBusinessPortalSpec struct {
	// Comp-business-portal image version to deploy.
	Version string `json:"version"`
}

// CompTripShareSpec defines settings for comp-trip-share.
type CompTripShareSpec struct {
	// Comp-trip-share image version to deploy.
	Version string `json:"version"`
}

// CompHwAuxSpec defines settings for comp-hw-aux.
type CompHwAuxSpec struct {
	// Comp-hw-aux image version to deploy.
	Version string `json:"version"`
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
	// Summon image version to deploy. If this isn't specified, AutoDeploy must be.
	// +optional
	Version string `json:"version,omitempty"`
	// Branch to watch for new images and auto-deploy.
	// +optional
	AutoDeploy string `json:"autoDeploy,omitempty"`
	// Name of the secret to use for secret values.
	Secrets []string `json:"secrets,omitempty"`
	// Name of the secret to use for image pulls. Defaults to `"pull-secret"`.
	// +optional
	PullSecret string `json:"pullSecret,omitempty"`
	// Summon-platform.yml configuration options.
	Config map[string]ConfigValue `json:"config,omitempty"`
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
	// Migration override settings.
	// +optional
	MigrationOverrides MigrationOverridesSpec `json:"migrationOverrides,omitempty"`
	// Celery settings.
	// +optional
	Celery CelerySpec `json:"celery,omitempty"`
	// Redis resource settings.
	// +optional
	Redis RedisSpec `json:"redis,omitempty"`
	// Pod replica settings.
	// +optional
	Replicas ReplicasSpec `json:"replicas,omitempty"`
	// Google Cloud project to use.
	// +optional
	GCPProject string `json:"gcpProject,omitempty"`
	// Toggle bools for enabling metrics exporting
	// +optional
	Metrics MetricsSpec `json:"metrics,omitempty"`
	// Enable monitoring
	// +optional
	Monitoring MonitoringSpec `json:"monitoring,omitempty"`
	// Enable mock car server
	// +optional
	EnableMockCarServer bool `json:"enableMockCarServer,omitempty"`
	// If Mock car server enabled, provide tenant hardware type
	// +optional
	// +kubebuilder:validation:Enum=OTAKEYS,MENSA
	MockTenantHardwareType string `json:"mockTenantHardwareType,omitempty"`
	// Settings for comp-dispatch.
	// +optional
	Dispatch CompDispatchSpec `json:"dispatch,omitempty"`
	// Settings for comp-business-portal.
	// +optional
	BusinessPortal CompBusinessPortalSpec `json:"businessPortal,omitempty"`
	// Settings for comp-trip-share.
	// +optional
	TripShare CompTripShareSpec `json:"tripShare,omitempty"`
	// Settings for comp-hw-aux.
	// +optional
	HwAux CompHwAuxSpec `json:"hwAux,omitempty"`
	// Feature flag to disable the CORE-1540 fixup in case it goes AWOL.
	// To be removed when support for the 1540 fixup is removed in summon.
	// +optional
	NoCore1540Fixup bool `json:"noCore1540Fixup,omitempty"`
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
// +kubebuilder:resource:shortName=summon;sp
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="summon version deployed"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="object status"
type SummonPlatform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SummonPlatformSpec   `json:"spec,omitempty"`
	Status SummonPlatformStatus `json:"status,omitempty"`
}

type summonPlatform interface {
	IsAutoscaled(component string) bool
}

func (summonplatform SummonPlatform) IsAutoscaled(component string) bool {
	switch component {
	case "celeryd":
		return summonplatform.Spec.Replicas.CelerydAuto
	/* TODO: fill in later when we expand out
	case "businessPortal":
		return summonplatform.Spec.Replicas.BuisnessPortalAuto
	case "channelworker":
		return summonplatform.Spec.Replicas.ChannelWorkerAuto
	*/
	default:
		return false
	}
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
