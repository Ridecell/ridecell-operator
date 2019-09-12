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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type extensionsComponent struct{}

func NewExtensions() *extensionsComponent {
	return &extensionsComponent{}
}

func (comp *extensionsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.PostgresExtension{},
	}
}

func (_ *extensionsComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)
	return instance.Status.DatabaseStatus == dbv1beta1.StatusReady
}

func (_ *extensionsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)
	existingStatus := map[string]string{}

	for extName, extVersion := range instance.Spec.Extensions {
		// Make the template extras.
		extras := map[string]interface{}{}
		extras["ExtensionName"] = extName
		extras["ExtensionVersion"] = extVersion
		extConn := instance.Status.AdminConnection.DeepCopy()
		extConn.Database = instance.Spec.DatabaseName
		extras["ExtensionConn"] = extConn

		res, _, err := ctx.CreateOrUpdate("extension.yml.tpl", extras, func(goalObj, existingObj runtime.Object) error {
			goal := goalObj.(*dbv1beta1.PostgresExtension)
			existing := existingObj.(*dbv1beta1.PostgresExtension)
			// Copy the Spec over.
			existing.Spec = goal.Spec
			// Grab the status.
			existingStatus[extName] = existing.Status.Status
			return nil
		})
		if err != nil {
			return res, errors.Wrapf(err, "Error from %s extension", extName)
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.PostgresDatabase)
		instance.Status.ExtensionStatus = existingStatus
		instance.Status.Status = dbv1beta1.StatusCreating
		return nil
	}}, nil
}
