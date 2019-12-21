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

// RDSSnapshotSpec defines the desired state of RDSSnapshot
type RDSSnapshotSpec struct {
	// +kubebuilder:validation:MinLength=2
	RDSInstanceID string `json:"rdsInstanceID"`
	//+kubebuilder:validation:Pattern=^[a-zA-Z][a-zA-Z0-9-]*[a-zA-Z0-9]$
	// +optional
	SnapshotID string `json:"snapshotID,omitempty"`
	// TTL is the time until the object cleans itself up
	// +optional
	TTL metav1.Duration `json:"ttl,omitempty"`
}

// RDSSnapshotStatus defines the observed state of RDSSnapshot
type RDSSnapshotStatus struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	SnapshotID string `json:"snapshotId"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RDSSnapshot is the Schema for the RDSs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status",description="object status"
type RDSSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RDSSnapshotSpec   `json:"spec,omitempty"`
	Status RDSSnapshotStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RDSSnapshotList contains a list of RDSSnapshot
type RDSSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RDSSnapshot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RDSSnapshot{}, &RDSSnapshotList{})
}
