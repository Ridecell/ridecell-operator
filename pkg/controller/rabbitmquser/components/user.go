/*
Copyright 2018-2019 Ridecell, Inc.

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
	"github.com/Ridecell/ridecell-operator/pkg/utils"
	"github.com/michaelklishin/rabbit-hole"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
)

type userComponent struct {
	Client utils.NewTLSClientFactory
}

func RabbitholeTLSClientFactory(uri string, user string, pass string, t *http.Transport) (utils.RabbitMQManager, error) {
	return rabbithole.NewTLSClient(uri, user, pass, t)
}

func (comp *userComponent) InjectFakeNewTLSClient(fakeFunc utils.NewTLSClientFactory) {
	comp.Client = fakeFunc
}

func NewUser() *userComponent {
	return &userComponent{Client: RabbitholeTLSClientFactory}
}

func (_ *userComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *userComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *userComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RabbitmqUser)

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
	xs, err := rmqc.ListUsers()
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error connecting or fetching rabbitmq users")
	}

	var user_exists bool
	for _, user := range xs {
		if user.Name == instance.Spec.Username {
			user_exists = true
		}
	}
	if !user_exists {
		var userPassword string
		userPassword, err = instance.Spec.PasswordSecretref.Resolve(ctx, "rabbitmqUserPassword")
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error resolving rabbitmq user password")
		}
		resp, _ := rmqc.PutUser(instance.Spec.Username, rabbithole.UserSettings{Password: userPassword, Tags: instance.Spec.Tags})
		if resp.StatusCode != 201 {
			return components.Result{}, errors.Wrapf(err, "unable to create rabbitmq user %s", instance.Spec.Username)
		}
	}
	return components.Result{}, nil
}
