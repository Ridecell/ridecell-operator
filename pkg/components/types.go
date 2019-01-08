/*
Copyright 2018 Ridecell, Inc.

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

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
}

// A ComponentContext is the state for a single reconcile request to the controller.
type ComponentContext struct {
	client.Client
	templates http.FileSystem
	Context   context.Context // This should probably go away
	Top       runtime.Object
	Scheme    *runtime.Scheme
}

// A component is a Promise Theory actor inside a controller.
type Component interface {
	WatchTypes() []runtime.Object
	IsReconcilable(*ComponentContext) bool
	Reconcile(*ComponentContext) (reconcile.Result, error)
}

// GathererComponent enables the accumulator pattern on Reconciler.
type GathererComponent interface {
	Component
	// WatchPredicateFuncs determines if a handler.MapObject should trigger a
	// reconcile request inferred from Top's metav1.Object interface.
	//
	// Note: Reconcile request are limited to those inferrable by Top's
	// metav1.Object by the NewReconciler in order to forbid components to
	// generate side effects other than by their Reconcile function itself.
	//
	// Insofar, the semantics are different from it's underlying
	// EnqueueRequestsFromMapFunc while implementing a superset of
	// EnqueueRequestForOwner semantics. You should avoid to reimplement the
	// latter through this function.
	WatchPredicateFuncs() predicate.Funcs
}

type Status interface{}

type Statuser interface {
	GetStatus() Status
	SetStatus(Status)
	SetErrorStatus(string)
}
