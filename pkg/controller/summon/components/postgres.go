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
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
)

type postgresComponent struct{}

func NewPostgres() *postgresComponent {
	return &postgresComponent{}
}

func (comp *postgresComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.PostgresDatabase{},
	}
}

func (_ *postgresComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *postgresComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	var existing *dbv1beta1.PostgresDatabase
	res, _, err := ctx.CreateOrUpdate("postgres_database.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.PostgresDatabase)
		existing = existingObj.(*dbv1beta1.PostgresDatabase)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	if existing != nil {
		// If the database is in an error state, mark this summon as error'd too.
		if existing.Status.Status == dbv1beta1.StatusError {
			return res, errors.Errorf("postgres: %s", existing.Status.Message)
		}
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.PostgresStatus = existing.Status.Status
			instance.Status.PostgresConnection = existing.Status.Connection
			if existing.Status.Status != "" {
				instance.Status.Status = summonv1beta1.StatusInitializing
			}
			return nil
		}
	}
	return res, err
}
