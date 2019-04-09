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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
	"github.com/michaelklishin/rabbit-hole"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

type vhostComponent struct {
	Client utils.NewTLSClientFactory
}

func (comp *vhostComponent) InjectFakeNewTLSClient(fakeFunc utils.NewTLSClientFactory) {
	comp.Client = fakeFunc
}

func NewVhost() *vhostComponent {
	return &vhostComponent{Client: utils.RabbitholeTLSClientFactory}
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
	rmqc, err := utils.OpenRabbit(ctx, &instance.Spec.Connection, comp.Client)
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
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RabbitmqVhost)
		instance.Status.Status = dbv1beta1.StatusReady
		instance.Status.Message = fmt.Sprintf("Vhost %s ready", instance.Spec.VhostName)
		return nil
	}}, nil
}
