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
	"crypto/tls"
	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/michaelklishin/rabbit-hole"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
)

type vhostComponent struct{}

func NewVhost() *vhostComponent {
	return &vhostComponent{}
}

func (_ *vhostComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *vhostComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *vhostComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RabbitmqVhost)

	transport := &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true, // test server certificate is not trusted in case of self hosted rabbitmq
	},
	}
	// Connect to the rabbitmq cluster
	rmqc, err := rabbithole.NewTLSClient(instance.Spec.ClusterHost, instance.Spec.Connection.Username, instance.Spec.Connection.PasswordSecretRef.Key, transport)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Create New TLS Client")
	}
	// Create the required vhost if it does not exist
	xs, err := rmqc.ListVhosts()
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Get all rabbitmq Vhosts")
	}

	var vhost_exists bool
	for _, element := range xs {
		if element.Name == instance.Spec.VhostName {
			vhost_exists = true
		}
	}
	if !vhost_exists {
		resp, _ := rmqc.PutVhost(instance.Spec.VhostName, rabbithole.VhostSettings{Tracing: false})
		if resp.StatusCode != 201 {
			return components.Result{}, errors.Wrapf(err, "vhost: unable to create vhost %s", instance.Spec.VhostName)
		}
	}
	return components.Result{}, nil
}
