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

// TODO: This whole thing should probably be its own custom resource.

import (
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type superuserComponent struct{}

func NewSuperuser() *superuserComponent {
	return &superuserComponent{}
}

func (comp *superuserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&summonv1beta1.DjangoUser{},
	}
}

func (comp *superuserComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// Wait for the database to be up and migrated.
	if instance.Status.PostgresStatus != dbv1beta1.StatusReady {
		return false
	}
	if instance.Status.Status == summonv1beta1.StatusReady {
		return true
	}
	if instance.Status.Status != summonv1beta1.StatusDeploying {
		return false
	}
	return true
}

func (comp *superuserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	if instance.Spec.NoCreateSuperuser {
		// Make sure the object doesn't exist.
		user, err := ctx.GetTemplate("superuser.yml.tpl", nil)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "unable to load superuser.yml.tpl template")
		}
		err = ctx.Delete(ctx.Context, user)
		if err != nil && !kerrors.IsNotFound(err) {
			return components.Result{Requeue: true}, errors.Wrap(err, "unable to delete superuser")
		}
		return components.Result{}, nil
	}

	res, _, err := ctx.CreateOrUpdate("superuser.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*summonv1beta1.DjangoUser)
		existing := existingObj.(*summonv1beta1.DjangoUser)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}
