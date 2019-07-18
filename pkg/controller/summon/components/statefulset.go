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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type statefulsetComponent struct {
	templatePath    string
	waitForDatabase bool
}

func NewStatefulSet(templatePath string, waitForDatabase bool) *statefulsetComponent {
	return &statefulsetComponent{templatePath: templatePath, waitForDatabase: waitForDatabase}
}

func (comp *statefulsetComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&appsv1.StatefulSet{},
	}
}

func (comp *statefulsetComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// Check on the pull secret. Not technically needed in some cases, but just wait.
	if instance.Status.PullSecretStatus != secretsv1beta1.StatusReady {
		return false
	}
	// If we don't need the database, we're ready.
	if !comp.waitForDatabase {
		return true
	}
	// We do want the database, so check all the database statuses.
	if instance.Status.PostgresStatus != dbv1beta1.StatusReady {
		return false
	}
	if instance.Status.Status != summonv1beta1.StatusDeploying {
		return false
	}
	return true
}

func (comp *statefulsetComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*appsv1.StatefulSet)
		existing := existingObj.(*appsv1.StatefulSet)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
