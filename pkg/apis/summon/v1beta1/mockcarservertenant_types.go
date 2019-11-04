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
	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MockCarServerTenantSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Enum=OTAKEYS,MENSA
	TenantHardwareType string `json:"tenantHardwareType"`
	// Callback url for Mock Tenant
	CallbackUrl string `json:"callbackUrl"`
	// +optional
	SkipFinalizers bool `json:"skipFinalizers,omitempty"`
}

type MockCarServerTenantStatus struct {
	Status        string `json:"status,omitempty"`
	Message       string `json:"message,omitempty"`
	KeysSecretRef string `json:"keyssecretref,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type MockCarServerTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MockCarServerTenantSpec   `json:"spec,omitempty"`
	Status MockCarServerTenantStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MockCarServerTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MockCarServerTenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MockCarServerTenant{}, &MockCarServerTenantList{})
}
