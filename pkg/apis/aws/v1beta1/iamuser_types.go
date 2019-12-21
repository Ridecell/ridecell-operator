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

// IAMUserSpec defines the desired state of IAMUser
type IAMUserSpec struct {
	UserName               string            `json:"username,omitempty"`
	InlinePolicies         map[string]string `json:"inlinePolicies,omitempty"`
	PermissionsBoundaryArn string            `json:"permissionsBoundaryArn"`
}

// IAMUserStatus defines the observed state of IAMUser
type IAMUserStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMUser is the Schema for the IAMUsers API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="object status"
type IAMUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IAMUserSpec   `json:"spec,omitempty"`
	Status IAMUserStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IAMUserList contains a list of IAMUser
type IAMUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IAMUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IAMUser{}, &IAMUserList{})
}
