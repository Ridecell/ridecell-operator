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

package cloudamqp_firewall_rules

import (
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	cfrcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/cloudamqp_firewall_rules/components"
	corev1 "k8s.io/api/core/v1"
)

// The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	_, err := components.NewReconciler("cloudamqp_firewall_rules-controller", mgr, &corev1.Node{}, nil, []components.Component{
		cfrcomponents.NewCloudamqpFirewallRule(),
	})
	return err
}
