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
	"crypto/rand"
	"encoding/base64"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type secretComponent struct{}

func NewSecret() *secretComponent {
	return &secretComponent{}
}

func (_ *secretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *secretComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *secretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var secretName string
	res, _, err := ctx.CreateOrUpdate("secret.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		// Store the name for the status output.
		secretName = existing.Name
		// Create a password if needed.
		val, ok := existing.Data["password"]
		if !ok || len(val) == 0 {
			rawPassword := make([]byte, 32)
			rand.Read(rawPassword)
			password := make([]byte, base64.RawURLEncoding.EncodedLen(32))
			base64.RawURLEncoding.Encode(password, rawPassword)
			existing.Data["password"] = password
		}
		return nil
	})
	res.StatusModifier = func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RDSInstance)
		instance.Status.Connection.PasswordSecretRef.Name = secretName
		instance.Status.Connection.PasswordSecretRef.Key = "password"
		return nil
	}
	return res, err
}
