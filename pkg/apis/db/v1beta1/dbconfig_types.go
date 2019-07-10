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

package v1beta1

import (
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A copy of postgresv1.PostgresParam, see below for why. This time it's the
// Parameters field that is incorrectly required.
type LocalPostgresqlParam struct {
	PgVersion  string            `json:"version"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// This is basically a subset of the fields of postgresv1.PostgresSpec which we care about.
// We cannot use PostgresSpec directly because it doesn't use the same validation tagging
// as controller-tools expects and c-t automatically picks it up as a substruct.
type LocalPostgresSpec struct {
	PostgresqlParam LocalPostgresqlParam `json:"postgresql,omitempty"`
	Volume          postgresv1.Volume    `json:"volume,omitempty"`
	// Patroni         postgresv1.Patroni         `json:"patroni,omitempty"`
	Resources postgresv1.Resources `json:"resources,omitempty"`

	DockerImage string `json:"dockerImage,omitempty"`

	// vars that enable load balancers are pointers because it is important to know if any of them is omitted from the Postgres manifest
	// in that case the var evaluates to nil and the value is taken from the operator config
	EnableMasterLoadBalancer  *bool `json:"enableMasterLoadBalancer,omitempty"`
	EnableReplicaLoadBalancer *bool `json:"enableReplicaLoadBalancer,omitempty"`

	// load balancers' source ranges are the same for master and replica services
	AllowedSourceRanges []string `json:"allowedSourceRanges,omitempty"`

	NumberOfInstances    int32                           `json:"numberOfInstances.omitempty"`
	Users                map[string]postgresv1.UserFlags `json:"users,omitempty"`
	MaintenanceWindows   []postgresv1.MaintenanceWindow  `json:"maintenanceWindows,omitempty"`
	Clone                postgresv1.CloneDescription     `json:"clone,omitempty"`
	Databases            map[string]string               `json:"databases,omitempty"`
	Tolerations          []corev1.Toleration             `json:"tolerations,omitempty"`
	Sidecars             []postgresv1.Sidecar            `json:"sidecars,omitempty"`
	InitContainers       []corev1.Container              `json:"init_containers,omitempty"`
	PodPriorityClassName string                          `json:"pod_priority_class_name,omitempty"`
	ShmVolume            *bool                           `json:"enableShmVolume,omitempty"`
}

type PostgresDbConfig struct {
	// +kubebuilder:validation:Enum=Exclusive,Shared
	Mode  string             `json:"mode"`
	RDS   *RDSInstanceSpec   `json:"rds,omitempty"`
	Local *LocalPostgresSpec `json:"local,omitempty"`
}

// DbConfigSpec defines the desired state of DbConfig
type DbConfigSpec struct {
	Postgres PostgresDbConfig `json:"postgres"`
	// Create a periscope user for postgres db. Defaults to true.
	// +optional
	NoCreatePeriscopeUser bool `json:"noCreatePeriscopeUser,omitempty"`
	// TODO RabbitMQ stuff too.
}

type PostgresDbConfigStatus struct {
	Status     string             `json:"status"`
	Connection PostgresConnection `json:"connection"`
}

// DbConfigStatus defines the observed state of DbConfig
type DbConfigStatus struct {
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	Postgres      PostgresDbConfigStatus `json:"postgres"`
	RDSInstanceID string                 `json:"rdsInstanceId,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DbConfig is the Schema for the DbConfigs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type DbConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DbConfigSpec   `json:"spec,omitempty"`
	Status DbConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DbConfigList contains a list of DbConfig
type DbConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DbConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DbConfig{}, &DbConfigList{})
}
