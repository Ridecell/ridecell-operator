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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"math/rand"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

type secretsComponent struct{}

func NewSecrets() *secretsComponent {
	return &secretsComponent{}
}

func (_ *secretsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *secretsComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *secretsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RabbitmqUser)
	secretName := fmt.Sprintf("%s.rabbitmq-user-password", instance.Name)

	// Generate password for user
	userPassword := RandStringRunes(20)
	target := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: instance.Namespace},
		Data: map[string][]byte{
			"password": []byte(userPassword),
		},
	}
	err := controllerutil.SetControllerReference(instance, target, ctx.Scheme)
	if err != nil {
		return components.Result{Requeue: true}, err
	}
	err = ctx.Update(ctx.Context, target)
	if err != nil && kerrors.IsNotFound(err) {
		err = ctx.Create(ctx.Context, target)
	}
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "secret: unable to save secret %s/%s", instance.Namespace, secretName)
	}

	return components.Result{}, nil
}

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	var alphaNumList = []rune("123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = alphaNumList[rand.Intn(len(alphaNumList))]
	}
	return string(b)
}
