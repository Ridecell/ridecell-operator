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
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_cloudamqp"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cloudamqpFirewallRuleComponent struct {
}

func NewCloudamqpFirewallRule() *cloudamqpFirewallRuleComponent {
	return &cloudamqpFirewallRuleComponent{}
}

func (_ *cloudamqpFirewallRuleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *cloudamqpFirewallRuleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *cloudamqpFirewallRuleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	// Get cloudamqp api key
	if os.Getenv("CLOUDAMQP_API_KEY") == "" {
		glog.Errorf("CLOUDAMQP_FIREWALL: No CLOUDAMQP_API_KEY found or variable is empty.")
		return components.Result{}, nil
	}
	apiKey := os.Getenv("CLOUDAMQP_API_KEY")

	apiUrl := "https://api.cloudamqp.com/api/security/firewall"
	// check for server url in env variable
	if os.Getenv("CLOUDAMQP_API_URL") != "" {
		apiUrl = os.Getenv("CLOUDAMQP_API_URL")
	}

	var ipList []string
	var data []utils.Rule

	if os.Getenv("CLOUDAMQP_FIREWALL") != "true" {
		// Add allow_all rule
		data = append(data, utils.Rule{
			IP:          "0.0.0.0/0",
			Services:    []string{"AMQP", "AMQPS"},
			Description: "Allow All",
		})
		ipList = append(ipList, "0.0.0.0/0")
	} else {
		// Get all Node objects
		nodeList := &corev1.NodeList{}
		// For unit tests, get node list from test function
		if os.Getenv("CLOUDAMQP_TEST") == "true" {
			nodeList = fake_cloudamqp.GetTestNodeList()
		} else {
			err := ctx.List(ctx.Context, &client.ListOptions{}, nodeList)
			if err != nil {
				glog.Errorf("CLOUDAMQP_FIREWALL: failed to list Node objects")
				return components.Result{RequeueAfter: time.Second * 15}, nil
			}
		}

		//--- add allow_all rule for now - will be removed after successful testing
		data = append(data, utils.Rule{
			IP:          "0.0.0.0/0",
			Services:    []string{"AMQP", "AMQPS"},
			Description: "Allow All",
		})
		//---

		nodeIP := ""
		// Iterate over Node items and add public IP to rule list
		for _, node := range nodeList.Items {
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeExternalIP {
					ipList = append(ipList, address.Address)
					nodeIP = address.Address
				}
			}
			if nodeIP != "" {
				data = append(data, utils.Rule{
					IP:          fmt.Sprintf("%s/32", nodeIP),
					Services:    []string{"AMQP", "AMQPS"},
					Description: "K8s Cluster Node IP",
				})
			}
			nodeIP = ""
		}
	}
	glog.Infof("CLOUDAMQP_FIREWALL: Whitelisted IPs: %s", ipList)

	// apply the IP rules to CLOUDAMQP FIREWALL
	err := utils.PutCloudamqpFirewallRules(apiUrl, apiKey, data)
	if err != nil {
		glog.Errorf("CLOUDAMQP_FIREWALL: failed to marshal json data")
		return components.Result{RequeueAfter: time.Second * 15}, nil
	}
	// apply the IP rules to CLOUDAMQP FIREWALL
	err = utils.PutCloudamqpFirewallRules(apiUrl, apiKey, payloadBytes)
	if err != nil {
		glog.Errorf("CLOUDAMQP_FIREWALL: failed to put firewall rules: %s ", err)
	}
	glog.Infof("CLOUDAMQP_FIREWALL: firewall rules updated")

	return components.Result{}, nil
}
