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
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secretComponent struct{}

func NewSecret() *secretComponent {
	return &secretComponent{}
}

func (comp *secretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
	}
}

func (_ *secretComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (_ *secretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)

	// If namespace is not defaulted and does not match no action is needed
	if instance.Spec.DbConfigRef.Namespace != "" && instance.Spec.DbConfigRef.Namespace != instance.Namespace {
		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(ctx.Context, types.NamespacedName{
			Name:      instance.Status.AdminConnection.PasswordSecretRef.Name,
			Namespace: instance.Spec.DbConfigRef.Namespace,
		}, fetchSecret)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "secret: unable to get dbconfig secret")
		}

		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fetchSecret.Name,
				Namespace: instance.Namespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: fetchSecret.Data,
		}

		_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, newSecret, func(existingObj runtime.Object) error {
			existing := existingObj.(*corev1.Secret)
			// Copy over important bits
			existing.ObjectMeta = newSecret.ObjectMeta
			existing.Type = newSecret.Type
			existing.Data = newSecret.Data
			return nil
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "secret: failed to copy secret")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.PostgresDatabase)
		instance.Status.Status = dbv1beta1.StatusCreating
		return nil
	}}, nil
}
