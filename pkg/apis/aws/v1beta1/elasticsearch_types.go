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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ElasticSearchSpec defines the desired state of ElasticSearch
type ElasticSearchSpec struct {
	// +optional
	VPCID string `json:"vpcId,omitempty"`
	// +optional
	SubnetIds []string `json:"subnetIds,omitempty"`
	// +optional
	SecurityGroupId string `json:"securityGroupId,omitempty"`
	// +optional
	// +kubebuilder:validation:Enum=Production,Development
	DeploymentType string `json:"deploymentType,omitempty"`
	// +optional
	InstanceType string `json:"instanceType,omitempty"`
	// +optional
	NoOfInstances int64 `json:"noOfInstances,omitempty"`
	// +optional
	ElasticSearchVersion string `json:"elasticSearchVersion,omitempty"`
	// +optional
	StoragePerNode int64 `json:"storagePerNode,omitempty"`
	// +optional
	SnapshotTime string `json:"snapshotTime,omitempty"`
}

// ElasticSearchStatus defines the observed state of ElasticSearch
type ElasticSearchStatus struct {
	Status         string `json:"status"`
	Message        string `json:"message"`
	DomainEndpoint string `json:"domainEndpoint"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticSearch is the Schema for the ElasticSearchs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type ElasticSearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticSearchSpec   `json:"spec,omitempty"`
	Status ElasticSearchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticSearchList contains a list of ElasticSearch
type ElasticSearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticSearch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticSearch{}, &ElasticSearchList{})
}
