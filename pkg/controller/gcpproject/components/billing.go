/*
Copyright 2020 Ridecell, Inc.

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

	"golang.org/x/net/context"
	"google.golang.org/api/cloudbilling/v1"
	"k8s.io/apimachinery/pkg/runtime"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

// Interface for a firebase client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_cloudbilling_test.go . GCPCloudBilling
type GCPCloudBilling interface {
	GetProjectBillingInfo(string) (*cloudbilling.ProjectBillingInfo, error)
	UpdateProjectbillingInfo(string) (*cloudbilling.ProjectBillingInfo, error)
}

type realCloudBilling struct {
	svc *cloudbilling.APIService
}

func newRealCloudBilling() (*realCloudBilling, error) {
	svc, err := cloudbilling.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &realCloudBilling{svc: svc}, nil
}

func (r *realCloudBilling) GetProjectBillingInfo(projectID string) (*cloudbilling.ProjectBillingInfo, error) {
	return r.svc.Projects.GetBillingInfo(fmt.Sprintf("projects/%s", projectID)).Do()
}

func (r *realCloudBilling) UpdateProjectbillingInfo(projectID string) (*cloudbilling.ProjectBillingInfo, error) {
	newBillingInfo := &cloudbilling.ProjectBillingInfo{
		BillingAccountName: os.Getenv("GOOGLE_BILLING_ACCOUNT_NAME"),
	}
	return r.svc.Projects.UpdateBillingInfo(fmt.Sprintf("projects/%s", projectID), newBillingInfo).Do()
}

type billingComponent struct {
	billing GCPCloudBilling
}

func NewBilling() *billingComponent {
	var billing GCPCloudBilling
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		var err error
		billing, err = newRealCloudBilling()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}
	}

	return &billingComponent{billing: billing}
}

func (comp *billingComponent) InjectBilling(billing GCPCloudBilling) {
	comp.billing = billing
}

func (_ *billingComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *billingComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *billingComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.GCPProject)
	// All of this code makes me a bit sad if i'm honest
	if comp.billing == nil {
		return components.Result{}, errors.New("gcpproject: firebase credentials not available")
	}

	if os.Getenv("GOOGLE_BILLING_ACCOUNT_NAME") == "" {
		return components.Result{}, errors.New("gcpproject: google billing account name not available")
	}

	if instance.Spec.EnableBilling != nil && *instance.Spec.EnableBilling {
		billingInfo, err := comp.billing.GetProjectBillingInfo(instance.Spec.ProjectID)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "gcpproject: failed to get project billing info")
		}

		if billingInfo.BillingAccountName != os.Getenv("GOOGLE_BILLING_ACCOUNT_NAME") {
			_, err := comp.billing.UpdateProjectbillingInfo(instance.Spec.ProjectID)
			if err != nil {
				return components.Result{}, errors.Wrap(err, "gcpproject: failed to update billing info")
			}
		}
	}

	// Clear operation names if exists and move on
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*gcpv1beta1.GCPProject)
		instance.Status.Status = gcpv1beta1.StatusReady
		instance.Status.Message = "Ready"
		return nil
	}}, nil
}
