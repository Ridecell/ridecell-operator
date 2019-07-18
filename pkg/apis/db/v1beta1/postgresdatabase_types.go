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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresDatabaseSpec defines the desired state of PostgresDatabase
type PostgresDatabaseSpec struct {
	// Name of the database to create. Defaults the same name as the PostgresDatabase object.
	// +optional
	DatabaseName string `json:"databaseName,omitempty"`
	// If enabled, do not automatically create the owner user.
	// +optional
	SkipUser bool `json:"skipUser,omitempty"`
	// Name of the user to own this database. Defaults to a user with the same name as `DatabaseName`.
	// +optional
	Owner string `json:"owner,omitempty"`
	// An optional ref to a DbConfig object to use for configuration. Defaults to the name of the namespace.
	// +optional
	DbConfigRef corev1.ObjectReference `json:"dbConfigRef,omitempty"`
	// A map of extensions to versions to install. If the version is "" then the latest version will be used.
	// +optional
	Extensions map[string]string `json:"extensions,omitempty"`
}

// PostgresDatabaseStatus defines the observed state of PostgresDatabase
type PostgresDatabaseStatus struct {
	Status                string             `json:"status"`
	Message               string             `json:"message"`
	DatabaseClusterStatus string             `json:"databaseClusterStatus"`
	DatabaseStatus        string             `json:"databaseStatus"`
	ExtensionStatus       map[string]string  `json:"extensionStatus,omitempty"`
	UserStatus            string             `json:"userStatus"`
	Connection            PostgresConnection `json:"connection"`
	AdminConnection       PostgresConnection `json:"adminConnection"`
	RDSInstanceID         string             `json:"rdsInstanceId,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgresDatabase is the Schema for the PostgresDatabases API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type PostgresDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresDatabaseSpec   `json:"spec,omitempty"`
	Status PostgresDatabaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PostgresDatabaseList contains a list of PostgresDatabase
type PostgresDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresDatabase{}, &PostgresDatabaseList{})
}
