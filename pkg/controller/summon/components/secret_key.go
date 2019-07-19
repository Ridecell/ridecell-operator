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
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secretKeyComponent struct{}

func NewSecretKey() *secretKeyComponent {
	return &secretKeyComponent{}
}

func (comp *secretKeyComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
	}
}

func (_ *secretKeyComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	// No ability to return errors so we're doing checks for this in Reconcile
	return true
}

func (comp *secretKeyComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	fetchSecret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.secret-key", instance.Name), Namespace: instance.Namespace}, fetchSecret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			// Secret may or may not exist, unknown error occurs
			return components.Result{}, errors.Wrapf(err, "secret_key: Unable to get secret")
		}
	}

	// See if there is data in the secret
	val, ok := fetchSecret.Data["SECRET_KEY"]
	if ok && len(val) != 0 {
		return components.Result{}, nil
	}

	// Secret does not exist or has no data, make a new one.
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.secret-key", instance.Name), Namespace: instance.Namespace},
		Data:       map[string][]byte{},
	}

	// Generate random string
	rawKey := make([]byte, 64)
	rand.Read(rawKey)
	newKey := make([]byte, base64.RawStdEncoding.EncodedLen(64))
	base64.RawStdEncoding.Encode(newKey, rawKey)

	newSecretKeyMap := map[string][]byte{"SECRET_KEY": newKey}
	newSecret.Data = newSecretKeyMap

	existing := newSecret.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, existing, func() error {
		// Sync important fields.
		err := controllerutil.SetControllerReference(instance, existing, ctx.Scheme)
		if err != nil {
			return errors.Wrapf(err, "secret_key: Failed to set controller reference")
		}
		existing.Annotations = newSecret.Annotations
		existing.Labels = newSecret.Labels
		existing.Type = newSecret.Type
		existing.Data = newSecret.Data
		return nil
	})

	if err != nil {
		return components.Result{}, errors.Wrap(err, "secret_key: Failed to update secret")
	}

	return components.Result{}, nil
}
