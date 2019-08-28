/*
Copyright 2019 Ridecell, Inc.
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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type redisDeploymentComponent struct {
	templatePath string
}

func NewRedisDeployment(templatePath string) *redisDeploymentComponent {
	return &redisDeploymentComponent{templatePath: templatePath}
}

func (comp *redisDeploymentComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&appsv1.Deployment{},
	}
}

func (comp *redisDeploymentComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// Check on the pull secret. Not technically needed in some cases, but just wait.
	if instance.Status.PullSecretStatus != secretsv1beta1.StatusReady {
		return false
	}
	return true
}

func (comp *redisDeploymentComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// If we're not in deploying state do nothing and exit early.
	if instance.Status.Status != summonv1beta1.StatusDeploying {
		return components.Result{}, nil
	}
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*appsv1.Deployment)
		existing := existingObj.(*appsv1.Deployment)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
