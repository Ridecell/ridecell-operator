/*
Copyright 2019-2020 Ridecell, Inc.

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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type userComponent struct {
	ClientFactory utils.RabbitMQClientFactory
}

func (comp *userComponent) InjectClientFactory(factory utils.RabbitMQClientFactory) {
	comp.ClientFactory = factory
}

func NewUser() *userComponent {
	return &userComponent{ClientFactory: utils.RabbitholeClientFactory}
}

func (_ *userComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *userComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *userComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RabbitmqUser)
	secretName := fmt.Sprintf("%s.rabbitmq-user-password", instance.Name)

	// Connect to the rabbitmq cluster
	rmqc, err := utils.OpenRabbit(ctx, &instance.Spec.Connection, comp.ClientFactory)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error creating rabbitmq client")
	}

	secret := &corev1.Secret{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: secretName, Namespace: instance.Namespace}, secret)
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "rabbitmq: Unable to load password secret %s/%s", instance.Namespace, secretName)
	}
	userPassword, ok := secret.Data["password"]
	if !ok {
		return components.Result{Requeue: true}, errors.Errorf("rabbitmq: Password secret %s/%s has no key \"password\"", instance.Namespace, secretName)
	}

	resp, err := rmqc.PutUser(instance.Spec.Username, rabbithole.UserSettings{Password: string(userPassword), Tags: instance.Spec.Tags})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error connection to rabbitmq host")
	}
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return components.Result{}, errors.Wrapf(err, "unable to create rabbitmq user %s", instance.Spec.Username)
	}

	// Data for the status modifier.
	hostAndPort, err := utils.RabbitHostAndPort(rmqc)
	if err != nil {
		return components.Result{}, err
	}
	username := instance.Spec.Username

	// Good to go.
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RabbitmqUser)
		instance.Status.Status = dbv1beta1.StatusReady
		instance.Status.Message = fmt.Sprintf("User %s ready", username)
		instance.Status.Connection.Host = hostAndPort.Host
		instance.Status.Connection.Port = hostAndPort.Port
		instance.Status.Connection.Username = username
		instance.Status.Connection.PasswordSecretRef.Name = fmt.Sprintf("%s.rabbitmq-user-password", instance.Name)
		instance.Status.Connection.PasswordSecretRef.Key = "password"
		return nil
	}}, nil
}
