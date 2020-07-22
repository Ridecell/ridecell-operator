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
	var desiredRules []utils.CloudamqpFirewallRule

	if os.Getenv("CLOUDAMQP_FIREWALL") != "true" {
		// Add allow_all rule
		desiredRules = append(desiredRules, utils.CloudamqpFirewallRule{
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
				desiredRules = append(desiredRules, utils.CloudamqpFirewallRule{
					IP:          fmt.Sprintf("%s/32", nodeIP),
					Services:    []string{"AMQP", "AMQPS"},
					Description: "K8s Cluster Node IP",
				})
			}
			nodeIP = ""
		}
	}
	// Get firewall rules and check if it needs to be updated
	rules, err := utils.GetCloudamqpFirewallRules(apiUrl, apiKey)
	if err != nil {
		glog.Errorf("CLOUDAMQP_FIREWALL: failed to get firewall rules: %s ", err)
		return components.Result{RequeueAfter: time.Second * 15}, nil
	}

	var putFlag bool
	var found bool
	if len(desiredRules) == len(rules) {
		for _, dRule := range desiredRules {
			found = false
			for _, rule := range rules {
				if dRule.IP == rule.IP {
					found = true
					break
				}
			}
			if !found {
				putFlag = true
				break
			}
		}
	} else {
		putFlag = true
	}

	if putFlag {
		// apply the IP rules to CLOUDAMQP FIREWALL
		glog.Infof("CLOUDAMQP_FIREWALL: Whitelisting node IPs: %s", ipList)
		err := utils.PutCloudamqpFirewallRules(apiUrl, apiKey, desiredRules)
		if err != nil {
			glog.Errorf("CLOUDAMQP_FIREWALL: failed to put firewall rules: %s ", err)
			return components.Result{RequeueAfter: time.Second * 15}, nil
		}
		return components.Result{RequeueAfter: time.Second * 30}, nil
	}
	return components.Result{}, nil
}
