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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AlertManagerConfigSpec defines the desired state of AlertManagerConfig
type AlertManagerConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AlertManagerName      string   `json:"alertManagerName,omitempty"`
	AlertManagerNamespace string   `json:"alertMangerNamespace,omitempty"`
	Route                 string   `json:"routes,omitempty"`
	Receivers             []string `json:"receivers,omitempty"`
}

// AlertManagerConfigStatus defines the observed state of AlertManagerConfig
type AlertManagerConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status  string `json:"status"`
	Message string `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlertManagerConfig is the Schema for the alertmanagerconfigs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type AlertManagerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertManagerConfigSpec   `json:"spec,omitempty"`
	Status AlertManagerConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AlertManagerConfigList contains a list of AlertManagerConfig
type AlertManagerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlertManagerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlertManagerConfig{}, &AlertManagerConfigList{})
}
