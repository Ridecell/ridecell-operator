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
	"io/ioutil"
	"os"
	"time"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
	"github.com/michaelklishin/rabbit-hole"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

const RabbitmqUserFinalizer = "user.rabbitmq.finalizer"

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

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(RabbitmqUserFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(RabbitmqUserFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rabbitmquser: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(RabbitmqUserFinalizer, instance) {
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" && os.Getenv("ENABLE_FINALIZERS") == "true" {
				// Connect to the rabbitmq cluster
				rmqconn, err := utils.OpenRabbit(ctx, &instance.Spec.Connection, comp.ClientFactory)
				if err != nil {
					return components.Result{}, errors.Wrapf(err, "rabbitmquser: error creating rabbitmq client")
				}
				// delete user
				res, err := rmqconn.DeleteUser(instance.Spec.Username)
				if err != nil {
					return components.Result{}, errors.Wrapf(err, "rabbitmquser: error deleting rabbitmq user")
				}
				if res.StatusCode != 204 && res.StatusCode != 404 {
					return components.Result{}, errors.Errorf("rabbitmquser: unable to delete rabbitmq user %s: HTTP Status Code %s", instance.Spec.Username, string(res.StatusCode))
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(RabbitmqUserFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rabbitmquser: failed to update instance while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// Connect to the rabbitmq cluster
	rmqc, err := utils.OpenRabbit(ctx, &instance.Spec.Connection, comp.ClientFactory)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error creating rabbitmq client")
	}

	userPassword, err := instance.Status.Connection.PasswordSecretRef.Resolve(ctx, "password")
	if err != nil {
		return components.Result{}, errors.Wrap(err, "user: error getting user password")
	}

	resp, err := rmqc.PutUser(instance.Spec.Username, rabbithole.UserSettings{Password: string(userPassword), Tags: instance.Spec.Tags})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error connection to rabbitmq host")
	}
	if resp.StatusCode == 201 {
		// If this is the initial creation of the user reconcile again after 10 seconds
		// This is a hack to remedy amqp permissions being applied incorrectly immediately after creation.
		return components.Result{RequeueAfter: time.Second * 10}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error reading PutUser response: %s", resp.Status)
		}
		return components.Result{}, errors.Errorf("unable to create rabbitmq user %s: %s %s", instance.Spec.Username, resp.Status, body)
	}

	//Policies
	// Get all Permissions for a vhost, user. Add all mentioned in spec and Remove unwanted
	permInfo, err := rmqc.ListPermissionsOf(instance.Spec.Username)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "error listing permissions for user %s", instance.Spec.Username)
	}
	for key := range instance.Spec.Permissions {
		_, err := rmqc.UpdatePermissionsIn(instance.Spec.Permissions[key].Vhost, instance.Spec.Username, rabbithole.Permissions{
			Configure: instance.Spec.Permissions[key].Configure,
			Read:      instance.Spec.Permissions[key].Read,
			Write:     instance.Spec.Permissions[key].Write,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error creating / updating permissions for user %s and vhost %s", instance.Spec.Username, instance.Spec.Permissions[key].Vhost)
		}
		// Removes entries from the list of all permission that got updated
		for k := range permInfo {
			if permInfo[k].Vhost == instance.Spec.Permissions[key].Vhost {
				// Remove key from permInfo
				permInfo[k] = permInfo[len(permInfo)-1]
				permInfo = permInfo[:len(permInfo)-1]
				break
			}
		}
	}
	//Remove unwanted permissions
	for k := range permInfo {
		// 204 response code when permission is removed
		_, err := rmqc.ClearPermissionsIn(permInfo[k].Vhost, permInfo[k].User)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "error removing permissions for user %s and vhost %s", permInfo[k].User, permInfo[k].Vhost)
		}

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
		return nil
	}}, nil
}
