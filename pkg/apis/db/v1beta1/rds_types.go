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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RDSInstanceSpec defines the desired state of RDS
type RDSInstanceSpec struct {
	AllocatedStorage int64  `json:"allocatedStorage,omitempty"`
	DatabaseName     string `json:"databaseName,omitempty"`
	InstanceID       string `json:"instanceID,omitempty"`
	Engine           string `json:"engine,omitempty"`
	EngineVersion    string `json:"engineVersion,omitempty"`
	InstanceClass    string `json:"instanceClass,omitempty"`
	MultiAZ          *bool  `json:"multiAZ,omitempty"`
	//+kubebuilder:validation:Pattern=\D*:\d{2}:\d{2}-\D*:\d{2}:\d{2}
	MaintenanceWindow string            `json:"maintenanceWindow"`
	Parameters        map[string]string `json:"parameterOverrides,omitempty"`
	Username          string            `json:"username,omitempty"`
	SubnetGroupName   string            `json:"subnetGroupName,omitempty"`
	VPCID             string            `json:"vpcID,omitempty"`
}

// RDSInstanceStatus defines the observed state of RDSInstance
type RDSInstanceStatus struct {
	Status          string             `json:"status"`
	Message         string             `json:"message"`
	Connection      PostgresConnection `json:"rdsConnection"`
	InstanceID      string             `json:"instanceID"`
	SecurityGroupID string             `json:"securityGroupID"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RDS is the Schema for the RDSs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type RDSInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RDSInstanceSpec   `json:"spec,omitempty"`
	Status RDSInstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RDSInstanceList contains a list of RDSInstance
type RDSInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RDSInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RDSInstance{}, &RDSInstanceList{})
}
