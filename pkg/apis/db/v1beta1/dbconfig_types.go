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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PostgresDbConfig struct {
	// +kubebuilder:validation:Enum=Exclusive,Shared
	Mode  string                   `json:"mode"`
	RDS   *RDSInstanceSpec         `json:"rds,omitempty"`
	Local *postgresv1.PostgresSpec `json:"local,omitempty"`
}

// DbConfigSpec defines the desired state of DbConfig
type DbConfigSpec struct {
	Postgres PostgresDbConfig `json:"postgres"`
	// TODO RabbitMQ stuff too.
}

type PostgresDbConfigStatus struct {
	Status     string             `json:"status"`
	Connection PostgresConnection `json:"connection"`
}

// DbConfigStatus defines the observed state of DbConfig
type DbConfigStatus struct {
	Status   string                 `json:"status"`
	Message  string                 `json:"message"`
	Postgres PostgresDbConfigStatus `json:"postgres"`
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

// Helper method to fill in some defaults on a local DB PostgresSpec.
func (p *PostgresDbConfig) LocalWithDefaults(namespace string) *postgresv1.PostgresSpec {
	withDefaults := p.Local.DeepCopy()
	if withDefaults.TeamID == "" {
		withDefaults.TeamID = namespace
	}
	if withDefaults.NumberOfInstances == 0 {
		withDefaults.NumberOfInstances = 2
	}
	if withDefaults.Users == nil {
		withDefaults.Users = map[string]postgresv1.UserFlags{
			"ridecell-admin": postgresv1.UserFlags{"superuser"},
		}
	}
	return withDefaults
}
