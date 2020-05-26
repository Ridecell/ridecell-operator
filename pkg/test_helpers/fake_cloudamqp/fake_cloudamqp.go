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

package fake_cloudamqp

import (
	corev1 "k8s.io/api/core/v1"
)

// Faking the node list in unit tests
func GetTestNodeList() *corev1.NodeList {
	return &corev1.NodeList{
		Items: []corev1.Node{
			corev1.Node{
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						corev1.NodeAddress{
							Type:    corev1.NodeExternalIP,
							Address: "1.2.3.4",
						},
					},
				},
			},
		},
	}
}
