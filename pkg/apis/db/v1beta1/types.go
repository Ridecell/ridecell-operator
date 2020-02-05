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
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
)

const (
	StatusReady     = "Ready"
	StatusCreating  = "Creating"
	StatusModifying = "Modifying"
	StatusError     = "Error"
	StatusUnknown   = "Unknown"
	StatusSkipped   = "Skipped"
	StatusGranted   = "PermissionsGranted"
)

// Connection details for a Postgres database.
type PostgresConnection struct {
	Host              string            `json:"host"`
	Port              int               `json:"port,omitempty"`
	Username          string            `json:"username"`
	PasswordSecretRef helpers.SecretRef `json:"passwordSecretRef"`
	Database          string            `json:"database,omitempty"`
	SSLMode           string            `json:"sslmode,omitempty"`
}

// Currently unused but could hold future per-object connection details.
type RabbitmqConnection struct{}

type RabbitmqStatusConnection struct {
	Host              string            `json:"host"`
	Port              int               `json:"port,omitempty"`
	Username          string            `json:"username"`
	PasswordSecretRef helpers.SecretRef `json:"passwordSecretRef"`
	Vhost             string            `json:"vhost,omitempty"`
}

// Status information about cluster-level shared utility Postgres users.
type SharedUsersStatus struct {
	Periscope string `json:"periscope"`
	Reporting string `json:"reporting"`
}

// MigrationOverridesSpec defines value overrides used when migrating Ansible-based Summon instances into Kubernetes/ridecell-operator.
type MigrationOverridesSpec struct {
	// An optional RDS instance ID override. Only used if the DbConfig is in RDS mode. Exists for migrations from Ansible Summon.
	// +optional
	RDSInstanceID string `json:"rdsInstanceId,omitempty"`
	// An optional RDS master username override. Only used if the DbConfig is in RDS mode. Exists for migrations from Ansible Summon.
	// +optional
	RDSMasterUsername string `json:"rdsMasterUsername,omitempty"`
}
