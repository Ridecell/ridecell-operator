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
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/iam/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// Interface for an IAM client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_keymanager_test.go . KeyManager
type KeyManager interface {
	List(string, ...string) (*iam.ListServiceAccountKeysResponse, error)
	Create(string, *iam.CreateServiceAccountKeyRequest) (*iam.ServiceAccountKey, error)
	Delete(string) (*iam.Empty, error)
}

type realKeyManager struct {
	svc *iam.Service
}

func newRealKeyManager() (*realKeyManager, error) {
	svc, err := iam.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &realKeyManager{svc: svc}, nil
}

func (r *realKeyManager) List(name string, keyTypes ...string) (*iam.ListServiceAccountKeysResponse, error) {
	return r.svc.Projects.ServiceAccounts.Keys.List(name).KeyTypes(keyTypes...).Do()
}

func (r *realKeyManager) Create(name string, req *iam.CreateServiceAccountKeyRequest) (*iam.ServiceAccountKey, error) {
	return r.svc.Projects.ServiceAccounts.Keys.Create(name, req).Do()
}

func (r *realKeyManager) Delete(name string) (*iam.Empty, error) {
	return r.svc.Projects.ServiceAccounts.Keys.Delete(name).Do()
}

type keyComponent struct {
	km KeyManager
}

func NewKey() *keyComponent {
	var km KeyManager
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		var err error
		km, err = newRealKeyManager()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}
	}

	return &keyComponent{km: km}
}

func (comp *keyComponent) InjectKM(km KeyManager) {
	comp.km = km
}

func (_ *keyComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *keyComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *keyComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.ServiceAccount)

	projectPath := fmt.Sprintf("projects/%s", instance.Spec.Project)
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", instance.Spec.AccountName, instance.Spec.Project)
	serviceAccountPath := fmt.Sprintf("%s/serviceAccounts/%s", projectPath, serviceAccountEmail)

	secretExists := true
	fetchSecret := &corev1.Secret{}
	err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.gcp-credentials", instance.Name), Namespace: instance.Namespace}, fetchSecret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			secretExists = false
		} else {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to get secret")
		}
	}

	// Currently not deleting old keys
	// There is a default key that cannot be deleted
	// TODO: Find a way to differentiate the keys apart.
	if !secretExists {
		rb := &iam.CreateServiceAccountKeyRequest{}
		accountKey, err := comp.km.Create(serviceAccountPath, rb)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to create serviceaccount key")
		}

		jsonKey, err := base64.StdEncoding.DecodeString(accountKey.PrivateKeyData)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to decode base64 key")
		}

		extra := map[string]interface{}{}
		extra["serviceAccount"] = jsonKey
		_, _, err = ctx.CreateOrUpdate("secret.yml.tpl", extra, func(goalObj, existingObj runtime.Object) error {
			goal := goalObj.(*corev1.Secret)
			existing := existingObj.(*corev1.Secret)
			existing.Type = goal.Type
			existing.Data = goal.Data
			return nil
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to create secret")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*gcpv1beta1.ServiceAccount)
		instance.Status.Status = gcpv1beta1.StatusReady
		instance.Status.Message = "User exists and has secret"
		return nil
	}}, nil
}
