/*
Copyright 2018 Ridecell, Inc.

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

type RabbitmqPolicy struct {
	// Regular expression pattern used to match queues and exchanges,
	// , e.g. "^ha\..+"
	Pattern string `json:"pattern"`
	// What this policy applies to: "queues", "exchanges", etc.
	ApplyTo string `json:"apply-to,omitempty"`
	// Numeric priority of this policy.
	Priority int `json:"priority,omitempty"`
	// Additional arguments added to the entities (queues,
	// exchanges or both) that match a policy
	Definition string `json:"definition"`
}

// RabbitmqVhostSpec defines the desired state of RabbitmqVhost
type RabbitmqVhostSpec struct {
	VhostName  string                    `json:"vhostName,omitempty"`
	SkipUser   bool                      `json:"skipUser,omitempty"`
	Policies   map[string]RabbitmqPolicy `json:"policies,omitempty"`
	Connection RabbitmqConnection        `json:"connection,omitempty"`
}

// RabbitmqVhostStatus defines the observed state of RabbitmqVhost
type RabbitmqVhostStatus struct {
	Status     string                   `json:"status"`
	Message    string                   `json:"message"`
	Connection RabbitmqStatusConnection `json:"connection,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RabbitmqVhost is the Schema for the RabbitmqVhosts API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type RabbitmqVhost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RabbitmqVhostSpec   `json:"spec,omitempty"`
	Status RabbitmqVhostStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RabbitmqVhostList contains a list of RabbitmqVhost
type RabbitmqVhostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RabbitmqVhost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RabbitmqVhost{}, &RabbitmqVhostList{})
}
