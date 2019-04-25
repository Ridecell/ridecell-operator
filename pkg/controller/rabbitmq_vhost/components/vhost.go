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

package components

import (
	"fmt"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

type vhostComponent struct {
	ClientFactory utils.RabbitMQClientFactory
}

func (comp *vhostComponent) InjectClientFactory(factory utils.RabbitMQClientFactory) {
	comp.ClientFactory = factory
}

func NewVhost() *vhostComponent {
	return &vhostComponent{ClientFactory: utils.RabbitholeClientFactory}
}

func (_ *vhostComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *vhostComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *vhostComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RabbitmqVhost)

	// Connect to the rabbitmq cluster
	rmqc, err := utils.OpenRabbit(ctx, &instance.Spec.Connection, comp.ClientFactory)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error creating rabbitmq client")
	}

	// Create the required vhost if it does not exist
	xs, err := rmqc.ListVhosts()
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error connecting or fetching rabbitmq vhosts")
	}

	var vhost_exists bool
	for _, element := range xs {
		if element.Name == instance.Spec.VhostName {
			vhost_exists = true
			break
		}
	}
	if !vhost_exists {
		resp, err := rmqc.PutVhost(instance.Spec.VhostName, rabbithole.VhostSettings{})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error creating vhost %s", instance.Spec.VhostName)
		}
		if resp.StatusCode != 201 {
			return components.Result{}, errors.Errorf("unable to create vhost %s, got response code %v", instance.Spec.VhostName, resp.StatusCode)
		}
	}

	// Policies
	policiesList, err := rmqc.ListPoliciesIn(instance.Spec.VhostName)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error fetching policies for vhost %s", instance.Spec.VhostName)
	}
	for policyName := range instance.Spec.Policies {
		var pFound bool
		var index int
		actualPolicyName := fmt.Sprintf("%s-%s", instance.Spec.VhostName, policyName)
		for i := range policiesList {
			if actualPolicyName == policiesList[i].Name {
				pFound = true
				index = i
				break
			}
		}
		if pFound {
			// Remove policy that was found from the list
			policiesList[index] = policiesList[len(policiesList)-1]
			policiesList = policiesList[:len(policiesList)-1]
		}
		// Create/Update policy
		policy := rabbithole.Policy{}
		policy.Pattern = instance.Spec.Policies[policyName].Pattern
		policy.ApplyTo = instance.Spec.Policies[policyName].ApplyTo
		policy.Priority = instance.Spec.Policies[policyName].Priority
		policy.Definition = instance.Spec.Policies[policyName].PolicyDefinition
		_, err = rmqc.PutPolicy(instance.Spec.VhostName, actualPolicyName, policy)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error updating policy for vhost %s", instance.Spec.VhostName)
		}
	}
	// Remove policies for a vhost which are not in the Spec
	for i := range policiesList {
		_, err = rmqc.DeletePolicy(instance.Spec.VhostName, policiesList[i].Name)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error deleting policy %s for vhost %s", policiesList[i].Name, instance.Spec.VhostName)
		}
	}

	// Unless we aren't making a user, wait for it to be ready.
	var user *dbv1beta1.RabbitmqUser
	if !instance.Spec.SkipUser {
		user = &dbv1beta1.RabbitmqUser{}
		err = ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, user)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error fetching RabbitmqUser %s/%s", instance.Namespace, instance.Name)
		}
		if user.Status.Status != dbv1beta1.StatusReady {
			// Could make a specific status for this, but it shouldn't take long.
			return components.Result{}, nil
		}
	}

	// Data for the status modifier.
	hostAndPort, err := utils.RabbitHostAndPort(rmqc)
	if err != nil {
		return components.Result{}, err
	}
	vhostName := instance.Spec.VhostName

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RabbitmqVhost)
		instance.Status.Status = dbv1beta1.StatusReady
		instance.Status.Message = fmt.Sprintf("Vhost %s ready", instance.Spec.VhostName)
		instance.Status.Connection.Host = hostAndPort.Host
		instance.Status.Connection.Port = hostAndPort.Port
		if user != nil {
			instance.Status.Connection.Username = user.Status.Connection.Username
			instance.Status.Connection.PasswordSecretRef = user.Status.Connection.PasswordSecretRef
		}
		instance.Status.Connection.Vhost = vhostName
		return nil
	}}, nil
}
