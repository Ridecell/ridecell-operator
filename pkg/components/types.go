/*
Copyright 2018-2019 Ridecell, Inc.

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
	"context"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// // A componentReconciler is the data for a single reconciler. These are our
// side of the controller.
type componentReconciler struct {
	name       string
	top        runtime.Object
	templates  http.FileSystem
	components []Component
	client     client.Client
	manager    manager.Manager
	Controller controller.Controller
}

// A ComponentContext is the state for a single reconcile request to the controller.
type ComponentContext struct {
	client.Client
	templates http.FileSystem
	Context   context.Context // This should probably go away
	Top       runtime.Object
	Scheme    *runtime.Scheme
}

// A function which modifies component status.
type StatusModifier func(runtime.Object) error

// An extension of reconcile.Result with extra component-related fields.
type Result struct {
	// Requeue tells the Controller to requeue the reconcile key.  Defaults to false.
	Requeue bool
	// RequeueAfter if greater than 0, tells the Controller to requeue the reconcile key after the Duration.
	RequeueAfter time.Duration
	// An optional anonymous function to change the object status.
	StatusModifier StatusModifier
}

// A component is a Promise Theory actor inside a controller.
type Component interface {
	WatchTypes() []runtime.Object
	IsReconcilable(*ComponentContext) bool
	Reconcile(*ComponentContext) (Result, error)
}

// An optional interface for Components which want to receive errors.
type ErrorHandler interface {
	ReconcileError(*ComponentContext, error) (Result, error)
}

// An optional interface for Components which want to use EnqueueRequestsFromMapFunc instead of watching owned objects.
type MapFuncWatcher interface {
	WatchMap(handler.MapObject, client.Client) ([]reconcile.Request, error)
}

// Opaque type for some kind of status substruct.
type Status interface{}

// Interface that all top-level objects must implement to work with the component system.
type Statuser interface {
	GetStatus() Status
	SetStatus(Status)
	SetErrorStatus(string)
}
