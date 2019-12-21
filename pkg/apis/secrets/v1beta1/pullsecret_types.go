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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PullSecretSpec defines the desired state of PullSecret
type PullSecretSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	// Name of the secret to use for image pulls. Defaults to `"pull-secret"`.
	// +optional
	PullSecretName string `json:"pullSecretName,omitempty"`
}

// PullSecretStatus defines the observed state of PullSecret
type PullSecretStatus struct {
	// Overall object status
	Status string `json:"status,omitempty"`

	// Message related to the current status.
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PullSecret is the Schema for the secrets API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="object status"
type PullSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PullSecretSpec   `json:"spec,omitempty"`
	Status PullSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PullSecretList contains a list of PullSecret
type PullSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PullSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PullSecret{}, &PullSecretList{})
}
