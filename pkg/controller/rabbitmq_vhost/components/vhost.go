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

type vhostComponent struct {
	Client NewTLSClientFactory
}

type RabbitMQManager interface {
	ListVhosts() ([]rabbithole.VhostInfo, error)
	PutVhost(string, rabbithole.VhostSettings) (*http.Response, error)
}

type NewTLSClientFactory func(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error)

func RabbitholeTLSClientFactory(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error) {
	return rabbithole.NewTLSClient(uri, user, pass, t)
}

func (comp *vhostComponent) InjectFakeNewTLSClient(fakeFunc NewTLSClientFactory) {
	comp.Client = fakeFunc
}

func NewVhost() *vhostComponent {
	return &vhostComponent{Client: RabbitholeTLSClientFactory}
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
		InsecureSkipVerify: instance.Spec.Connection.InsecureSkip,
	},
	}

	hostPassword, err := instance.Spec.Connection.Password.Resolve(ctx, "password")

	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error resolving rabbitmq connection credentials")
	}

	// Connect to the rabbitmq cluster
	rmqc, err := comp.Client(instance.Spec.Connection.Host, instance.Spec.Connection.Username, hostPassword, transport)

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
		}
	}
	if !vhost_exists {
		resp, _ := rmqc.PutVhost(instance.Spec.VhostName, rabbithole.VhostSettings{Tracing: false})
		if resp.StatusCode != 201 {
			return components.Result{}, errors.Wrapf(err, "unable to create vhost %s", instance.Spec.VhostName)
		}
	}
	return components.Result{}, nil
}
