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
	"fmt"
	"net/http"
	"reflect"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func NewReconciler(name string, mgr manager.Manager, top runtime.Object, templates http.FileSystem, components []Component) (*componentReconciler, error) {
	cr := &componentReconciler{
		name:       name,
		top:        top,
		templates:  templates,
		components: components,
		manager:    mgr,
	}

	// Create the controller.
	c, err := controller.New(name, mgr, controller.Options{Reconciler: cr})
	if err != nil {
		return nil, fmt.Errorf("unable to create controller: %v", err)
	}
	cr.Controller = c

	// Watch for changes in the Top object.
	err = c.Watch(&source.Kind{Type: cr.top}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return nil, fmt.Errorf("unable to create top-level watch: %v", err)
	}

	// Watch for changes in other objects.
	watchedTypes := map[reflect.Type]bool{}
	for _, comp := range cr.components {
		for _, watchObj := range comp.WatchTypes() {
			var watchHandler handler.EventHandler
			mfComp, isAMapFuncWatch := comp.(MapFuncWatcher)
			if isAMapFuncWatch {
				// Watch an arbitrary object via a MapFunc.
				watchHandler = &handler.EnqueueRequestsFromMapFunc{
					ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
						requests, err := mfComp.WatchMap(obj, cr.client)
						if err != nil {
							// For lack anything better to do for now ...
							fmt.Printf("ERROR FROM MAP FUNC: %s", err)
							panic(err)
						}
						return requests
					}),
				}
			} else {
				// Watch an owned object, but first check if we're already watching this type.
				watchType := reflect.TypeOf(watchObj).Elem()
				_, ok := watchedTypes[watchType]
				if ok {
					// Already watching.
					continue
				}
				watchedTypes[watchType] = true
				watchHandler = &handler.EnqueueRequestForOwner{
					IsController: true,
					OwnerType:    cr.top,
				}
			}

			err = c.Watch(&source.Kind{Type: watchObj}, watchHandler)
			if err != nil {
				return nil, errors.Wrap(err, "unable to create watch")
			}
		}
	}

	return cr, nil
}

func (cr *componentReconciler) newContext(request reconcile.Request) (*ComponentContext, error) {
	reqCtx := context.TODO()

	// Fetch the current value of the top object for this reconcile.
	top := cr.top.DeepCopyObject()
	err := cr.client.Get(reqCtx, request.NamespacedName, top)
	if err != nil {
		return nil, err
	}

	ctx := &ComponentContext{
		templates: cr.templates,
		Context:   reqCtx,
		Top:       top,
	}
	err = cr.manager.SetFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("error calling manager.SetFields: %v", err)
	}
	return ctx, nil
}

func (cr *componentReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.Infof("[%s] %s: Reconciling!", request.NamespacedName, cr.name)

	// Build a reconciler context to pass around.
	ctx, err := cr.newContext(request)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Top object not found, likely already deleted.
			return reconcile.Result{}, nil
		}
		// Some other fetch error, try again on the next tick.
		return reconcile.Result{Requeue: true}, err
	}

	// Make a clean copy of the top object to diff against later. This is used for
	// diffing because the status subresource might not always be available.
	cleanTop := ctx.Top.DeepCopyObject()

	// Reconcile all the components.
	// start := time.Now()
	result, err := cr.reconcileComponents(ctx)
	// fmt.Printf("$$$ Reconcile took %s\n", time.Since(start))
	if err != nil {
		// fmt.Printf("@@@@ Reconcile error %v\n", err)
		ctx.Top.(Statuser).SetErrorStatus(err.Error())
	}

	// Check if an update to the status subresource is required.
	if !reflect.DeepEqual(ctx.Top.(Statuser).GetStatus(), cleanTop.(Statuser).GetStatus()) {
		// Update the top object status.
		glog.V(2).Infof("[%s] Reconcile: Updating Status\n", request.NamespacedName)
		err = cr.modifyStatus(ctx, result.statusModifiers)
		if err != nil {
			result.result.Requeue = true
			return result.result, err
		}
	}

	return result.result, nil
}

// A holding struct for the overall result of a reconcileComponents call.
type reconcilerResults struct {
	// The current context.
	ctx *ComponentContext
	// The pending result to return from the Reconcile.
	result reconcile.Result
	// All the status modifier functions to replay in case of a write collision.
	statusModifiers []StatusModifier
	// The most recent error.
	err error
}

func (r *reconcilerResults) mergeResult(componentResult Result, component Component, err error) error {
	if err != nil {
		r.err = err
	}
	if componentResult.Requeue {
		r.result.Requeue = true
	}
	if componentResult.RequeueAfter != 0 && (r.result.RequeueAfter == 0 || r.result.RequeueAfter > componentResult.RequeueAfter) {
		r.result.RequeueAfter = componentResult.RequeueAfter
	}
	if componentResult.StatusModifier != nil {
		r.statusModifiers = append(r.statusModifiers, componentResult.StatusModifier)
		statusErr := componentResult.StatusModifier(r.ctx.Top)
		if statusErr != nil {
			instance := r.ctx.Top.(metav1.Object)
			glog.Errorf("[%s/%s] Error running status modifier from %#v: %s\n", instance.GetNamespace(), instance.GetName(), component, statusErr)
			if r.err == nil {
				// If we already had a real error, don't mask it, otherwise propagate this error.
				err = errors.Wrap(statusErr, "Error running initial status modifier")
				r.err = err
			}
		}
	}
	return err
}

func (cr *componentReconciler) reconcileComponents(ctx *ComponentContext) (*reconcilerResults, error) {
	instance := ctx.Top.(metav1.Object)
	ready := []Component{}
	for _, component := range cr.components {
		glog.V(10).Infof("[%s/%s] reconcileComponents: Checking if %#v is available to reconcile", instance.GetNamespace(), instance.GetName(), component)
		if component.IsReconcilable(ctx) {
			glog.V(9).Infof("[%s/%s] reconcileComponents: %#v is available to reconcile", instance.GetNamespace(), instance.GetName(), component)
			ready = append(ready, component)
		}
	}
	res := &reconcilerResults{ctx: ctx}
	for _, component := range ready {
		// fmt.Printf("### Reconciling %#v\n", component)
		// start := time.Now()
		innerRes, err := component.Reconcile(ctx)
		// fmt.Printf("### Done reconciling %#v, took %s\n", component, time.Since(start))
		// Update result. This should be checked before the err!=nil because sometimes
		// we want to requeue immediately on error.
		err = res.mergeResult(innerRes, component, err)
		if err != nil {
			for _, errComponent := range ready {
				errReconciler, ok := errComponent.(ErrorHandler)
				if !ok {
					// Not an error handler, push on.
					continue
				}
				innerRes, errorErr := errReconciler.ReconcileError(ctx, err)
				res.mergeResult(innerRes, errComponent, nil)
				if errorErr != nil {
					// Can't really do much more than log it, sigh. Some day this should set a prometheus metric.
					glog.Errorf("[%s/%s] Error running error handler %#v: %s", instance.GetNamespace(), instance.GetName(), errComponent, errorErr)
				}
			}
			return res, err
		}
	}
	return res, nil
}

func (cr *componentReconciler) modifyStatus(ctx *ComponentContext, statusModifiers []StatusModifier) error {
	// Try for the fast path of a single save using the subresource
	err := ctx.Status().Update(ctx.Context, ctx.Top)
	if err == nil {
		// No error, fast path success!
		return nil
	}

	// Something went wrong so we have to do a re-get an apply of the modifiers.
	for tries := 0; tries < 5; tries++ {
		err = cr.updateStatus(ctx, ctx.Top, func(instance runtime.Object) error {
			for _, mod := range statusModifiers {
				err := mod(instance)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err == nil {
			// Success!
			return nil
		}
		// Leave err set so we can wrap the final error below.
	}

	instanceObj := ctx.Top.(metav1.Object)
	return errors.Wrapf(err, "unable to update status for %s/%s, too many failures", instanceObj.GetNamespace(), instanceObj.GetName())
}

func (cr *componentReconciler) updateStatus(ctx *ComponentContext, instance runtime.Object, mutateFn func(runtime.Object) error) error {
	// Get a fresh copy to replay changes against.
	instanceObj := instance.(metav1.Object)
	freshCopy := instance.DeepCopyObject()
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instanceObj.GetName(), Namespace: instanceObj.GetNamespace()}, freshCopy)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Object was deleted already, don't keep retrying, just ignore the error and move on.
			// This is kind of questionable, hopefully we don't regret it in the future.
			return nil
		}
		return errors.Wrapf(err, "error getting %s/%s for object status", instanceObj.GetNamespace(), instanceObj.GetName())
	}

	// Do stuff.
	err = mutateFn(freshCopy)
	if err != nil {
		return errors.Wrap(err, "error running status modifier")
	}

	// Try to save again, first with new API and then with old.
	err = ctx.Status().Update(ctx.Context, freshCopy)
	if err != nil && kerrors.IsNotFound(err) {
		err = ctx.Update(ctx.Context, freshCopy)
	}
	if err != nil {
		return errors.Wrapf(err, "error updating %s/%s for object status", instanceObj.GetNamespace(), instanceObj.GetName())
	}
	return nil
}

// componentReconciler implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &componentReconciler{}

// InjectClient injects the client.
func (v *componentReconciler) InjectClient(c client.Client) error {
	v.client = c
	return nil
}
