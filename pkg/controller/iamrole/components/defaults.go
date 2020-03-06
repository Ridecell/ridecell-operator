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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
)

type defaultsComponent struct {
}

func NewDefaults() *defaultsComponent {
	return &defaultsComponent{}
}

func (_ *defaultsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *defaultsComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *defaultsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.IAMRole)

	// Fill in defaults.
	if instance.Spec.RoleName == "" {
		instance.Spec.RoleName = instance.Name
	}

	if instance.Spec.PermissionsBoundaryArn == "" {
		defaultPermissionsBoundaryArn := os.Getenv("DEFAULT_PERMISSIONS_BOUNDARY_ARN")
		if defaultPermissionsBoundaryArn == "" {
			return components.Result{}, errors.New("iam_role: DEFAULT_PERMISSIONS_BOUNDARY_ARN is not set")
		}
		instance.Spec.PermissionsBoundaryArn = defaultPermissionsBoundaryArn
	}

	if instance.Spec.AssumeRolePolicyDocument == "" {
		trustedArnList := os.Getenv("TRUSTED_ROLE_ARNS")
		if trustedArnList == "" {
			return components.Result{}, errors.New("iam_role: TRUSTED_ROLE_ARNS is not set")
		}

		newStrings := strings.Split(trustedArnList, ",")
		// Sort the arns for safety
		sort.Strings(newStrings)

		arnJSONList, err := json.Marshal(newStrings)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: unable to marshal trusted arns into json")
		}

		defaultPolicy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": %s
					},
					"Action": "sts:AssumeRole"
				}
			]
		}`, string(arnJSONList))

		instance.Spec.AssumeRolePolicyDocument = defaultPolicy
	}

	return components.Result{}, nil
}
