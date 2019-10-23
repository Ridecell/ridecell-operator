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
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"math/rand"
	"os"
	"time"
)

const letterBytes = "1234567890abcdefghijklmnopqrstuvwxyz"

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
	instance := ctx.Top.(*summonv1beta1.MockCarServerTenant)
	var secretName string
	res, _, err := ctx.CreateOrUpdate("secret.yml.tpl", nil, func(_goalObj, existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		// Store the name for the status output.
		secretName = existing.Name
		// Create OTA keys, if needed.
		val, ok := existing.Data["OTAKEYS_API_KEY"]
		if !ok || len(val) == 0 {
			existing.Data["OTAKEYS_API_KEY"] = []byte(instance.Name + "-api-key")
		}
		val, ok = existing.Data["OTAKEYS_SECRET_KEY"]
		if !ok || len(val) == 0 {
			existing.Data["OTAKEYS_SECRET_KEY"] = RandStringBytes(32)
		}
		val, ok = existing.Data["OTAKEYS_TOKEN"]
		if !ok || len(val) == 0 {
			existing.Data["OTAKEYS_TOKEN"] = RandStringBytes(32)
		}
		val, ok = existing.Data["OTAKEYS_PUSH_API_KEY"]
		if !ok || len(val) == 0 {
			existing.Data["OTAKEYS_PUSH_API_KEY"] = RandStringBytes(32)
		}
		val, ok = existing.Data["OTAKEYS_PUSH_SECRET_KEY"]
		if !ok || len(val) == 0 {
			existing.Data["OTAKEYS_PUSH_SECRET_KEY"] = RandStringBytes(32)
		}
		val, ok = existing.Data["OTAKEYS_PUSH_TOKEN"]
		if !ok || len(val) == 0 {
			existing.Data["OTAKEYS_PUSH_TOKEN"] = RandStringBytes(32)
		}
		existing.Data["OTAKEYS_BASE_URL"] = []byte(os.Getenv("MOCKCARSERVER_URI") + "/otakeys/")
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Unable to create secret")
	}
	res.StatusModifier = func(obj runtime.Object) error {
		instance := obj.(*summonv1beta1.MockCarServerTenant)
		instance.Status.KeysSecretRef = secretName
		return nil
	}
	return res, err
}

func RandStringBytes(n int) []byte {
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[seededRand.Intn(len(letterBytes))]
	}
	return b
}
