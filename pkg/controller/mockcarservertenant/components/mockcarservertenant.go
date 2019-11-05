/*
Copyright 2019 Ridecell, Inc.

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
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const mockCarServerTenantFinalizer = "finalizer.mockcarservertenant.summon.ridecell.io"

type MockCarServerTenantComponent struct{}

func NewMockCarServerTenant() *MockCarServerTenantComponent {
	return &MockCarServerTenantComponent{}
}

func (_ *MockCarServerTenantComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *MockCarServerTenantComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *MockCarServerTenantComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.MockCarServerTenant)
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, instance)
	if err != nil {
		// Error reading the object - requeue the request.
		return components.Result{}, errors.Wrapf(err, "instance of MockCarServerTenant not found")
	}
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(mockCarServerTenantFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(mockCarServerTenantFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(mockCarServerTenantFinalizer, instance) {
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" {
				isDeleted, err := utils.DeleteMockTenant(instance.Name)
				if err != nil && !(isDeleted) {
					return components.Result{}, errors.Wrapf(err, "failed to delete MockCarServerTenant from server")
				}
				secret := &corev1.Secret{}
				err = ctx.Client.Get(ctx.Context, types.NamespacedName{Name: instance.Name + ".tenant-otakeys", Namespace: instance.Namespace}, secret)
				if err == nil {
					err = ctx.Delete(ctx.Context, secret)
					if err != nil {
						return components.Result{}, errors.Wrapf(err, "failed to delete MockCarServerTenant secret")
					}
				} else if err != nil && !k8serrors.IsNotFound(err) {
					return components.Result{}, errors.Wrapf(err, "failed to delete MockCarServerTenant secret")
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(mockCarServerTenantFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "failed to update MockCarServerTenant while removing finalizer")
			}
		}
		return components.Result{}, nil
	}
	// Get our password secret
	otakeysSecret := &corev1.Secret{}
	err = ctx.Client.Get(ctx.Context, types.NamespacedName{Name: instance.Name + ".tenant-otakeys", Namespace: instance.Namespace}, otakeysSecret)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "mockcarservertenant: failed to get otakeys secret")
	}
	// Get data from secret
	postData := map[string]string{}
	postData["api_key"] = string(otakeysSecret.Data["OTAKEYS_API_KEY"])
	postData["sec_key"] = string(otakeysSecret.Data["OTAKEYS_SECRET_KEY"])
	postData["api_token"] = string(otakeysSecret.Data["OTAKEYS_TOKEN"])
	postData["push_api_key"] = string(otakeysSecret.Data["OTAKEYS_PUSH_API_KEY"])
	postData["push_sec_key"] = string(otakeysSecret.Data["OTAKEYS_PUSH_SECRET_KEY"])
	postData["push_token"] = string(otakeysSecret.Data["OTAKEYS_PUSH_TOKEN"])

	// Add Name, hardware type, Callback url field
	postData["name"] = instance.Name
	postData["tenant_hardware_type"] = instance.Spec.TenantHardwareType
	postData["callback_url"] = instance.Spec.CallbackUrl
	// Create mock tenant
	isCreated, err := utils.CreateOrUpdateMockTenant(postData)
	if err != nil && !(isCreated) {
		return components.Result{}, errors.Wrapf(err, "mockcarservertenant: failed to create otakeys")
	}
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*summonv1beta1.MockCarServerTenant)
		instance.Status.Status = "Success"
		instance.Status.Message = "Mock car server tenant created."
		return nil
	}}, nil
}
