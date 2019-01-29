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

// DjangoUserSpec defines the desired state of DjangoUser
type DjangoUserSpec struct {
	Email          string             `json:"email"`
	PasswordSecret string             `json:"passwordSecret,omitempty"`
	Database       PostgresConnection `json:"database"`
	FirstName      string             `json:"firstName,omitempty"`
	LastName       string             `json:"lastName,omitempty"`
	Active         bool               `json:"active"`
	Manager        bool               `json:"manager"`
	Dispatcher     bool               `json:"dispatcher"`
	Staff          bool               `json:"staff"`
	Superuser      bool               `json:"superuser"`
}

// DjangoUserStatus defines the observed state of DjangoUser
type DjangoUserStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DjangoUser is the Schema for the djangousers API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type DjangoUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DjangoUserSpec   `json:"spec,omitempty"`
	Status DjangoUserStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DjangoUserList contains a list of DjangoUser
type DjangoUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DjangoUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DjangoUser{}, &DjangoUserList{})
}
