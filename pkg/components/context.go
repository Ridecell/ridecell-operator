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
	"fmt"
	"net/http"

	// "github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"

	"github.com/Ridecell/ridecell-operator/pkg/templates"
)

func (ctx *ComponentContext) GetTemplate(path string, extraData map[string]interface{}) (runtime.Object, error) {
	if ctx.templates == nil {
		return nil, fmt.Errorf("no templates loaded for this reconciler")
	}
	return templates.Get(ctx.templates, path, struct {
		Instance runtime.Object
		Extra    map[string]interface{}
	}{Instance: ctx.Top, Extra: extraData})
}

func (ctx *ComponentContext) CreateOrUpdate(path string, extraData map[string]interface{}, mutateFn func(runtime.Object, runtime.Object) error) (Result, controllerutil.OperationResult, error) {
	target, err := ctx.GetTemplate(path, extraData)
	if err != nil {
		return Result{}, controllerutil.OperationResultNone, err
	}
	fmt.Println(target)
	op, err := controllerutil.CreateOrUpdate(ctx.Context, ctx, target.DeepCopyObject(), func(existing runtime.Object) error {
		// Set owner ref.
		err := controllerutil.SetControllerReference(ctx.Top.(metav1.Object), existing.(metav1.Object), ctx.Scheme)
		if err != nil {
			return err
		}
		// Run the component-level mutator.
		err = mutateFn(target, existing)
		if err != nil {
			return err
		}
		// Sync the metadata fields.
		targetMeta := target.(metav1.ObjectMetaAccessor).GetObjectMeta().(*metav1.ObjectMeta)
		existingMeta := existing.(metav1.ObjectMetaAccessor).GetObjectMeta().(*metav1.ObjectMeta)
		return ReconcileMeta(targetMeta, existingMeta)
	})
	if err != nil {
		return Result{Requeue: true}, op, err
	}

	return Result{}, op, nil
}

// Make a copy of a context with new templates. Used mostly for shared components.
func (ctx *ComponentContext) WithTemplates(templates http.FileSystem) *ComponentContext {
	return &ComponentContext{
		Client:    ctx.Client,
		templates: templates,
		Context:   ctx.Context,
		Top:       ctx.Top,
		Scheme:    ctx.Scheme,
	}
}

// Method for creating a test context, for use in component unit tests.
func NewTestContext(top runtime.Object, templates http.FileSystem) *ComponentContext {
	// This method is ugly and I don't like it. I should rebuild this whole subsytem around interfaces and have an explicit fake for it.
	return &ComponentContext{
		Top:       top,
		Client:    fake.NewFakeClient(top),
		Scheme:    scheme.Scheme,
		templates: templates,
	}
}

// ComponentContext implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &ComponentContext{}

// InjectClient injects the client.
func (ctx *ComponentContext) InjectClient(c client.Client) error {
	ctx.Client = c
	return nil
}

// ComponentContext implements inject.Scheme.
// A scheme will be automatically injected.
var _ inject.Scheme = &ComponentContext{}

// InjectScheme injects the client.
func (ctx *ComponentContext) InjectScheme(s *runtime.Scheme) error {
	ctx.Scheme = s
	return nil
}
