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
	"encoding/json"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceAccountComponent struct {
	IAMSvc    *google.Service
	projectID string
}

func NewServiceAccount() *serviceAccountComponent {
	jsonKey := os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")
	if jsonKey == "" {
		return components.Result{}, errors.New("serviceaccount: service account json key is blank")
	}

	var serviceAccountJSON map[string]string
	err = json.Unmarshal([]byte(jsonKey), &serviceAccountJSON)
	if err != nil {
		return components.Result{}, errors.New("serviceaccount: failed to unmarshal service account json")
	}

	config, err := google.ConfigFromJSON([]byte(os.Getenv("GOOGLE_SERVICE_ACCOUNT_KEY")))
	if err != nil {
		return nil, errors.New("serviceaccount: failed to get config from json")
	}

	client := config.Client()

	svc, err := iam.New(client)
	if err != nil {
		return nil, errors.New("serviceaccount: failed to get iam service client")
	}

	return &serviceAccountComponent{
		comp.IAMSvc:    svc,
		comp.projectID: serviceAccountJSON["project_id"],
	}
}

func (comp *serviceAccountComponent) InjectGCPSvc(svc *google.Service, projectID string) {
	comp.IAMSvc = svc
	comp.projectID = projectID
	return nil
}

func (_ *serviceAccountComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *serviceAccountComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *serviceAccountComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.ServiceAccount)

	projectPath := fmt.Sprintf("projects/%s", comp.projectID)
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", instance.Spec.AccountName, comp.projectID)
	serviceAccountPath := fmt.Sprintf("%s/serviceAccounts/%s", projectPath, serviceAccountEmail)

	accountExists := true
	_, err = iamService.Projects.ServiceAccounts.Get(serviceAccountPath).Context(ctx).Do()
	if err != nil {
		if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
			accountExists = false
		} else {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to get serviceaccount")
		}
	}

	if !accountExists {
		serviceAccountRequest := &iam.CreateServiceAccountRequest{
			AccountId: instance.Spec.AccountName,
		}
		_, err := iamService.Projects.ServiceAccounts.Create(projectID, serviceAccountRequest).Context(ctx.Context).Do()
		if err != nil {
			return components.Result{}, errors.New("serviceaccount: failed to create service account")
		}
	}

	secretExists := true
	fetchSecret := &corev1.Secret{}
	err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.gcp-credentials", instance.Name), Namespace: instance.Namespace}, fetchSecret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			secretExists = false
		} else {
			return components.Result{}, errors.New("serviceaccount: failed to get secret")
		}
	}

	// Currently not deleting old keys
	// There is a default key that cannot be deleted
	// TODO: Find a way to differentiate the keys apart.
	if !secretExists {
		rb := &iam.CreateServiceAccountKeyRequest{}
		accountKey, err := iamService.Projects.ServiceAccounts.Keys.Create(serviceAccountPath, rb).Context(ctx).Do()
		if err != nil {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to create serviceaccount key")
		}

		jsonKey, err := base64.StdEncoding.DecodeString(accountKey.PrivateKeyData)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to decode base64 key")
		}

		extra := map[string]interface{}{}
		extra["serviceAccount"] = jsonKey
		_, _, err := ctx.CreateOrUpdate("secret.yml.tpl", extra, func(goalObj, existingObj runtime.Object) error {
			existing := existingObj.(*corev1.Secret)
			existing.Data = goalObj.Data
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
