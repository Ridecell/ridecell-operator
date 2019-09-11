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

// TODO: This whole thing should probably be its own custom resource.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// Cannot use ":" in k8s secret keys, so we've resorted to using this.
const CustomTimeLayout = "2006-01-02T15-04-05Z"

type fernetRotateComponent struct{}

func NewFernetRotate() *fernetRotateComponent {
	return &fernetRotateComponent{}
}

func (comp *fernetRotateComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
	}
}

func (_ *fernetRotateComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// The time comparisons done in the Reconcilable block below should be moved up here.
	return true
}

func (comp *fernetRotateComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// The following block checking the times of the keys should be in IsReconcilable, we cannot however return errors there so it is being moved here.
	fetchSecret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.fernet-keys", instance.Name), Namespace: instance.Namespace}, fetchSecret)

	// If secret doesn't exist reconcile to create new
	var secretFound bool
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return components.Result{}, errors.Wrapf(err, "rotate_fernet: Failed to retrieve default secrets object")
		}
	} else {
		secretFound = true
	}

	var latestTime time.Time
	for k := range fetchSecret.Data {
		parsedKey, err := time.Parse(CustomTimeLayout, k)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "rotate_fernet: Error while parsing time string")
		}
		if parsedKey.After(latestTime) {
			latestTime = parsedKey
		}
	}
	latestTimePlus := latestTime.Add(instance.Spec.FernetKeyLifetime)
	if !latestTimePlus.Before(time.Now().UTC()) {
		return components.Result{}, nil
	}

	// Generate new timeStamp string
	timeStamp := time.Time.Format(time.Now().UTC(), CustomTimeLayout)

	// Generate random string
	rawKey := make([]byte, 64)
	_, err = rand.Read(rawKey)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rotate_fernet: failed to write new key")
	}
	newKey := make([]byte, base64.RawStdEncoding.EncodedLen(64))
	base64.RawStdEncoding.Encode(newKey, rawKey)

	if !secretFound {
		fetchSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.fernet-keys", instance.Name), Namespace: instance.Namespace},
			Data:       map[string][]byte{},
		}
	}

	fetchSecret.Data[timeStamp] = newKey

	_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, fetchSecret, func(existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		// Sync important fields.
		err := controllerutil.SetControllerReference(instance, existing, ctx.Scheme)
		if err != nil {
			return errors.Wrapf(err, "rotate_fernet: Failed to set controller reference")
		}
		existing.ObjectMeta = fetchSecret.ObjectMeta
		existing.Type = fetchSecret.Type
		existing.Data = fetchSecret.Data
		return nil
	})

	if err != nil {
		return components.Result{}, errors.Wrap(err, "rotate_fernet: Failed to update secret")
	}

	return components.Result{}, nil
}
