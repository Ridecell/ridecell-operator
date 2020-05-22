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

package components

import (
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

type cloudamqpFirewallRuleComponent struct {
}

func NewCloudamqpFirewallRule() *cloudamqpFirewallRuleComponent {
	return &cloudamqpFirewallRuleComponent{}
}

func (_ *cloudamqpFirewallRuleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		//&corev1.Node{},
	}
}

func (_ *cloudamqpFirewallRuleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *cloudamqpFirewallRuleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	// Get cloudamqp api key
	// if os.Getenv("CLOUDAMQP_API_KEY") == "" {
	// 	glog.Errorf("CLOUDAMQP_FIREWALL: No CLOUDAMQP_API_KEY found or variable is empty.")
	// 	return components.Result{}, nil
	// }

	// Get all Node objects
	nodeList := &corev1.NodeList{}
	err := ctx.List(ctx.Context, &client.ListOptions{}, nodeList)
	if err != nil {
		glog.Errorf("CLOUDAMQP_FIREWALL: failed to list Node objects")
		return components.Result{}, nil
	}

	var nodePublicIPList []string

	for _, node := range nodeList.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeExternalIP {
				nodePublicIPList = append(nodePublicIPList, address.Address)
			}
		}
	}

	glog.Infof("CLOUDAMQP_FIREWALL: NodePublicIPs: %s", nodePublicIPList)

	return components.Result{}, nil
}
