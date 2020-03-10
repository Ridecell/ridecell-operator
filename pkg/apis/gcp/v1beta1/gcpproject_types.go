/*
Copyright 2020 Ridecell, Inc.

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

// GCPProjectSpec defines the desired state of GCPProject
type GCPProjectSpec struct {
	ProjectID string        `json:"projectID"`
	Parent    ProjectParent `json:"parent"`
}

// ProjectParent is used to populate cloudresourcemanager.ResourceId when project is created
type ProjectParent struct {
	// +kubebuilder:validation:Enum=organization,folder,project
	Type       string `json:"type"`
	ResourceID string `json:"resourceID"`
}

// GCPProjectStatus defines the observed state of GCPProject
type GCPProjectStatus struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	OperationName string `json:"operationName,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCPProject is the Schema for the GCPProjects API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type GCPProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GCPProjectSpec   `json:"spec,omitempty"`
	Status GCPProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GCPProjectList contains a list of GCPProject
type GCPProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GCPProject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GCPProject{}, &GCPProjectList{})
}
