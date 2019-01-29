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
	"crypto/rand"
	"encoding/base64"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type secretComponent struct{}

func NewSecret() *secretComponent {
	return &secretComponent{}
}

func (_ *secretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
	}
}

func (_ *secretComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *secretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.DjangoUser)

	existing := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Spec.PasswordSecret, Namespace: instance.Namespace}, existing)
	if err != nil && !kerrors.IsNotFound(err) {
		return components.Result{Requeue: true}, errors.Wrapf(err, "secret: unable to load secret %s/%s", instance.Namespace, instance.Spec.PasswordSecret)
	} else if err == nil {
		// Loaded correctly, if the password exists then we're done.
		val, ok := existing.Data["password"]
		if ok && len(val) > 0 {
			return components.Result{}, nil
		}
	}

	// If we got this far, we need to make a random password and save it. No this
	// is not double-base64-ing things.
	rawPassword := make([]byte, 16)
	rand.Read(rawPassword)
	password := make([]byte, base64.RawStdEncoding.EncodedLen(16))
	base64.RawStdEncoding.Encode(password, rawPassword)

	target := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: instance.Spec.PasswordSecret, Namespace: instance.Namespace},
		Data: map[string][]byte{
			"password": password,
		},
	}

	err = controllerutil.SetControllerReference(instance, target, ctx.Scheme)
	if err != nil {
		return components.Result{Requeue: true}, err
	}

	err = ctx.Update(ctx.Context, target)
	if err != nil && kerrors.IsNotFound(err) {
		err = ctx.Create(ctx.Context, target)
	}
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "secret: unable to save secret %s/%s", instance.Namespace, instance.Spec.PasswordSecret)
	}

	return components.Result{}, nil
}
