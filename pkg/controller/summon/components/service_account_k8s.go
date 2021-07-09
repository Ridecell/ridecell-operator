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
	"os"
	"regexp"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
)

type serviceAccountK8sComponent struct {
}

func NewserviceAccountK8s() *serviceAccountK8sComponent {
	return &serviceAccountK8sComponent{}
}

func (comp *serviceAccountK8sComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.ServiceAccount{},
	}
}

func (_ *serviceAccountK8sComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *serviceAccountK8sComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {

	//TODO Add better way to get accountID
	permissionsBoundaryArn := os.Getenv("PERMISSIONS_BOUNDARY_ARN")
	if permissionsBoundaryArn == "" {
		return components.Result{}, errors.Errorf("serviceaccountk8s: permissions_boundary_arn is empty")
	}
	match := regexp.MustCompile(`:([0-9]{6,}):`).FindStringSubmatch(permissionsBoundaryArn)
	if match == nil {
		return components.Result{}, errors.Errorf("serviceaccountk8s: unable to get account id from boundary arn")
	}

	extra := map[string]interface{}{}
	extra["accountId"] = match[1]

	res, _, err := ctx.CreateOrUpdate("service_account_k8s.yml.tpl", extra, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*corev1.ServiceAccount)
		existing := existingObj.(*corev1.ServiceAccount)
		existing.Annotations = goal.Annotations
		return nil
	})
	return res, err
}
