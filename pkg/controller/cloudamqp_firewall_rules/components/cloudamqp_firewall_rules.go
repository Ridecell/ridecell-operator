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
	"github.com/pkg/errors"
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
		&corev1.Node{},
	}
}

func (_ *cloudamqpFirewallRuleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *cloudamqpFirewallRuleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	// Get all Node objects
	nodeList := &corev1.NodeList{}
	err := ctx.List(ctx.Context, &client.ListOptions{}, nodeList)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "failed to list Node objects")
	}

	//var nodePublicIPList []string
	//var nodePrivateIPList []string
	var tmpPublicIp, tmpPrivateIp string

	for _, node := range nodeList.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeExternalIP {
				//nodePublicIPList = append(nodePublicIPList, address.Address)
				tmpPublicIp = address.Address
			}
			if address.Type == corev1.NodeInternalIP {
				//nodePrivateIPList = append(nodePrivateIPList, address.Address)
				tmpPrivateIp = address.Address
			}
		}
		glog.Infof("NodeIP: %s : %s", tmpPrivateIp, tmpPublicIp)
	}

	return components.Result{}, nil
}
