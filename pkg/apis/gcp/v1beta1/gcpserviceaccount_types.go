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

// ServiceAccountSpec defines the desired state of ServiceAccount
type GCPServiceAccountSpec struct {
	Project     string `json:"project"`
	AccountName string `json:"accountName,omitempty"`
	Description string `json:"description,omitempty"`
}

// GCPServiceAccountStatus defines the observed state of GCPServiceAccount
type GCPServiceAccountStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Email   string `json:"email"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCPServiceAccount is the Schema for the GCPServiceAccounts API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="object status"
type GCPServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GCPServiceAccountSpec   `json:"spec,omitempty"`
	Status GCPServiceAccountStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCPServiceAccountList contains a list of GCPServiceAccount
type GCPServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GCPServiceAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GCPServiceAccount{}, &GCPServiceAccountList{})
}
