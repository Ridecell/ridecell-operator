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
	"time"

	"golang.org/x/net/context"
	"google.golang.org/api/firebase/v1beta1"
	"google.golang.org/api/googleapi"
	"k8s.io/apimachinery/pkg/runtime"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

// Interface for a firebase client to allow for a mock implementation.
//go:generate moq -out zz_generated.mock_firebase_test.go . GCPFirebase
type GCPFirebase interface {
	Get(string) (*firebase.FirebaseProject, error)
	AddFirebase(string) (*firebase.Operation, error)
	GetOperation(string) (*firebase.Operation, error)
}

type realFirebase struct {
	svc *firebase.Service
}

func newRealFirebase() (*realFirebase, error) {
	svc, err := firebase.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &realFirebase{svc: svc}, nil
}

func (r *realFirebase) Get(projectID string) (*firebase.FirebaseProject, error) {
	return r.svc.Projects.Get(fmt.Sprintf("projects/%s", projectID)).Do()
}

func (r *realFirebase) GetOperation(name string) (*firebase.Operation, error) {
	return r.svc.Operations.Get(name).Do()
}

func (r *realFirebase) AddFirebase(projectID string) (*firebase.Operation, error) {
	return r.svc.Projects.AddFirebase(fmt.Sprintf("projects/%s", projectID), &firebase.AddFirebaseRequest{}).Do()
}

type firebaseProjectComponent struct {
	firebase GCPFirebase
}

func NewFirebaseProject() *firebaseProjectComponent {
	var firebase GCPFirebase
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		var err error
		firebase, err = newRealFirebase()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}
	}

	return &firebaseProjectComponent{firebase: firebase}
}

func (comp *firebaseProjectComponent) InjectFirebase(firebase GCPFirebase) {
	comp.firebase = firebase
}

func (_ *firebaseProjectComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *firebaseProjectComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *firebaseProjectComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.GCPProject)
	// All of this code makes me a bit sad if i'm honest
	if comp.firebase == nil {
		return components.Result{}, errors.New("gcpproject: firebase credentials not available")
	}

	if instance.Spec.EnableFirebase != nil && *instance.Spec.EnableFirebase {
		foundFirebase := true
		_, err := comp.firebase.Get(instance.Spec.ProjectID)
		if err != nil {
			// This doesn't throw 403 unless the GCP Project does not exist.
			if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
				foundFirebase = false
			} else {
				return components.Result{}, errors.Wrap(err, "gcpproject: failed to get firebase project")
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
		instance.Status.FirebaseOperationName = ""
		return nil
	}}, nil
}
