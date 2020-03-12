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
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/firebase/v1beta1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/runtime"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

// Interface for a cloudresourcemanager client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_cloudresourcemanager_test.go . GCPCloudResourceManager
type GCPCloudResourceManager interface {
	Get(string) (*cloudresourcemanager.Project, error)
	Create(*components.ComponentContext, string) (*cloudresourcemanager.Operation, error)
	GetOperation(string) (*cloudresourcemanager.Operation, error)
}

// Interface for a firebase client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_firebase_test.go . GCPFirebase
type GCPFirebase interface {
	Get(string) (*firebase.FirebaseProject, error)
	AddFirebase(string) (*firebase.Operation, error)
	GetOperation(string) (*firebase.Operation, error)
}

type realCloudResourceManager struct {
	svc *cloudresourcemanager.Service
}

type realFirebase struct {
	svc *firebase.Service
}

func newRealCloudResourceManager() (*realCloudResourceManager, error) {
	svc, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &realCloudResourceManager{svc: svc}, nil
}

func newRealFirebase() (*realFirebase, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: os.Getenv("MACHINE_USER_ACCESS_TOKEN"),
	})
	svc, err := firebase.NewService(context.Background(), option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	return &realFirebase{svc: svc}, nil
}

func (r *realCloudResourceManager) Get(projectID string) (*cloudresourcemanager.Project, error) {
	return r.svc.Projects.Get(projectID).Do()
}

func (r *realCloudResourceManager) Create(ctx *components.ComponentContext, projectID string) (*cloudresourcemanager.Operation, error) {
	instance := ctx.Top.(*gcpv1beta1.GCPProject)

	newProject := &cloudresourcemanager.Project{
		ProjectId: projectID,
		Parent: &cloudresourcemanager.ResourceId{
			Type: instance.Spec.Parent.Type,
			Id:   instance.Spec.Parent.ResourceID,
		},
	}
	return r.svc.Projects.Create(newProject).Do()
}

func (r *realCloudResourceManager) GetOperation(name string) (*cloudresourcemanager.Operation, error) {
	return r.svc.Operations.Get(name).Do()
}

func (r *realFirebase) Get(projectID string) (*firebase.FirebaseProject, error) {
	return r.svc.Projects.Get(projectID).Do()
}

func (r *realFirebase) GetOperation(name string) (*firebase.Operation, error) {
	return r.svc.Operations.Get(name).Do()
}

func (r *realFirebase) AddFirebase(projectID string) (*firebase.Operation, error) {
	return r.svc.Projects.AddFirebase(projectID, &firebase.AddFirebaseRequest{}).Do()
}

type gcpProjectComponent struct {
	crm      GCPCloudResourceManager
	firebase GCPFirebase
}

func NewProject() *gcpProjectComponent {
	var crm GCPCloudResourceManager
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		var err error
		crm, err = newRealCloudResourceManager()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}

	}

	var firebase GCPFirebase
	if os.Getenv("MACHINE_USER_ACCESS_TOKEN") != "" {
		var err error
		firebase, err = newRealFirebase()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}
	}

	return &gcpProjectComponent{crm: crm, firebase: firebase}
}

func (comp *gcpProjectComponent) InjectCRM(crm GCPCloudResourceManager) {
	comp.crm = crm
}

func (comp *gcpProjectComponent) InjectFirebase(firebase GCPFirebase) {
	comp.firebase = firebase
}

func (_ *gcpProjectComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *gcpProjectComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *gcpProjectComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.GCPProject)
	// All of this code makes me a bit sad if i'm honest
	if comp.crm == nil || comp.firebase == nil {
		return components.Result{}, errors.New("gcpproject: google credentials not available")
	}

	foundProject := true
	_, err := comp.crm.Get(instance.Spec.ProjectID)
	if err != nil {
		// Google appears to respond with a 403 when a project doesn't exist.
		// Catch 404 anyway just in case this assumption is wrong.
		if gErr, ok := err.(*googleapi.Error); ok && (gErr.Code == 404 || gErr.Code == 403) {
			foundProject = false
		} else {
			return components.Result{}, errors.Wrap(err, "gcpproject: failed to get project")
		}
	}

	if !foundProject {
		var operation *cloudresourcemanager.Operation
		// If we already tried to create one there should be a status stored
		var needsCreate bool
		// Storing and retrieving the operation name from status is not preferred
		// Google does not appear to expose operations.list for projects however
		if instance.Status.ProjectOperationName != "" {
			operation, err = comp.crm.GetOperation(instance.Status.ProjectOperationName)
			if err != nil {
				if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
					needsCreate = true
				} else {
					return components.Result{}, errors.Wrap(err, "gcpproject: failed to get operation")
				}
			}
		} else {
			needsCreate = true
		}

		if needsCreate {
			operation, err = comp.crm.Create(ctx, instance.Spec.ProjectID)
			if err != nil {
				return components.Result{}, errors.Wrap(err, "gcpproject: failed to create project")
			}
		}

		// If operation isn't done store the operation name in status and check back later
		if !operation.Done {
			return components.Result{
				StatusModifier: func(obj runtime.Object) error {
					instance := obj.(*gcpv1beta1.GCPProject)
					instance.Status.Message = "Waiting on project creation."
					instance.Status.ProjectOperationName = operation.Name
					return nil
				},
				RequeueAfter: time.Minute,
			}, nil
		}
	}

	if instance.Spec.EnableFirebase != nil && *instance.Spec.EnableFirebase {
		foundFirebase := true
		_, err := comp.firebase.Get(instance.Spec.ProjectID)
		if err != nil {
			if err != nil {
				// This doesn't throw 403 unless the GCP Project does not exist.
				if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
					foundFirebase = false
				} else {
					return components.Result{}, errors.Wrap(err, "gcpproject: failed to get firebase project")
				}
			}
		}

		if !foundFirebase {
			var firebaseOperation *firebase.Operation
			var firebaseNeedsCreate bool
			if instance.Status.FirebaseOperationName != "" {
				firebaseOperation, err = comp.firebase.GetOperation(instance.Status.FirebaseOperationName)
				if err != nil {
					if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
						firebaseNeedsCreate = true
					} else {
						return components.Result{}, errors.Wrap(err, "gcpproject: failed to get operation")
					}
				}
			} else {
				firebaseNeedsCreate = true
			}

			if firebaseNeedsCreate {
				firebaseOperation, err = comp.firebase.AddFirebase(instance.Spec.ProjectID)
				if err != nil {
					return components.Result{}, errors.Wrap(err, "gcpproject: failed to add firebase to project")
				}
			}

			if !firebaseOperation.Done {
				return components.Result{
					StatusModifier: func(obj runtime.Object) error {
						instance := obj.(*gcpv1beta1.GCPProject)
						instance.Status.Message = "Waiting on firebase creation."
						instance.Status.ProjectOperationName = ""
						instance.Status.FirebaseOperationName = firebaseOperation.Name
						return nil
					},
					RequeueAfter: time.Minute,
				}, nil
			}
		}
	}

	// Clear operation names if exists and move on
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*gcpv1beta1.GCPProject)
		instance.Status.Status = gcpv1beta1.StatusReady
		instance.Status.Message = "Ready"
		instance.Status.ProjectOperationName = ""
		instance.Status.FirebaseOperationName = ""
		return nil
	}}, nil
}
