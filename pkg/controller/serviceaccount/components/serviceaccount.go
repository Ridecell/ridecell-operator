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
	"fmt"
	"log"
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"
	"k8s.io/apimachinery/pkg/runtime"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
)

// Interface for an IAM client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_serviceaccountmanager_test.go . ServiceAccountManager
type ServiceAccountManager interface {
	Get(string) (*iam.ServiceAccount, error)
	Create(string, *iam.CreateServiceAccountRequest) (*iam.ServiceAccount, error)
}

type realServiceAccountManager struct {
	svc *iam.Service
}

func newRealServiceAccountManager() (*realServiceAccountManager, error) {
	svc, err := iam.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &realServiceAccountManager{svc: svc}, nil
}

func (r *realServiceAccountManager) Get(name string) (*iam.ServiceAccount, error) {
	return r.svc.Projects.ServiceAccounts.Get(name).Do()
}

func (r *realServiceAccountManager) Create(name string, req *iam.CreateServiceAccountRequest) (*iam.ServiceAccount, error) {
	return r.svc.Projects.ServiceAccounts.Create(name, req).Do()
}

type serviceAccountComponent struct {
	sam ServiceAccountManager
}

func NewServiceAccount() *serviceAccountComponent {
	var sam ServiceAccountManager
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		var err error
		sam, err = newRealServiceAccountManager()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}
	}

	return &serviceAccountComponent{sam: sam}
}

func (comp *serviceAccountComponent) InjectSAM(sam ServiceAccountManager) {
	comp.sam = sam
}

func (_ *serviceAccountComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *serviceAccountComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *serviceAccountComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.GCPServiceAccount)

	projectPath := fmt.Sprintf("projects/%s", instance.Spec.Project)
	serviceAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", instance.Spec.AccountName, instance.Spec.Project)
	serviceAccountPath := fmt.Sprintf("%s/serviceAccounts/%s", projectPath, serviceAccountEmail)

	accountExists := true
	_, err := comp.sam.Get(serviceAccountPath)
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
			ServiceAccount: &iam.ServiceAccount{
				Description: instance.Spec.Description,
			},
		}
		_, err := comp.sam.Create(projectPath, serviceAccountRequest)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "serviceaccount: failed to create service account")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*gcpv1beta1.GCPServiceAccount)
		instance.Status.Email = serviceAccountEmail
		return nil
	}}, nil
}
