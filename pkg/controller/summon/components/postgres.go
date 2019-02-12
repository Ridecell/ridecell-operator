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
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type postgresComponent struct {
	postgresTemplatePath string
	operatorTemplatePath string
}

func NewPostgres(postgresTemplatePath string, operatorTemplatePath string) *postgresComponent {
	return &postgresComponent{
		postgresTemplatePath: postgresTemplatePath,
		operatorTemplatePath: operatorTemplatePath,
	}
}

func (comp *postgresComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&postgresv1.Postgresql{},
		&dbv1beta1.PostgresOperatorDatabase{},
	}
}

func (_ *postgresComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *postgresComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	var res components.Result
	var db *postgresv1.Postgresql
	var err error
	if instance.Spec.Database.ExclusiveDatabase {
		res, db, err = comp.reconcileExclusiveDatabase(ctx, instance)
	} else {
		res, db, err = comp.reconcileOperatorDatabase(ctx, instance)
	}

	// Helper method to be used later.
	setPostgresStatus := func(obj runtime.Object) error {
		if db != nil {
			// db can be nil if an API call fails.
			instance.Status.PostgresStatus = db.Status
		}
		return nil
	}
	res.StatusModifier = setPostgresStatus

	if err != nil {
		// Hard error during the fetch or update.
		return res, err
	}

	// We got a database of some kind, time to work out the status.
	if !db.Status.Success() || db.Status == postgresv1.ClusterStatusInvalid {
		// One of the simple error cases like ClusterStatusUpdateFailed. For whatever reason, Invalid isn't checked in Success().
		// Not actually sure how the .Error field works. I think it's just ignored by the CRD entirely. Trying to play both sides just in case.
		if db.Error == "" {
			err = errors.Errorf("postgres: status is %s", db.Status)
		} else {
			err = errors.Errorf("postgres: status is %s: %s", db.Status, db.Error)
		}
		return res, err
	} else if db.Status != postgresv1.ClusterStatusUnknown {
		// In
		// DB creation was started at some point, and we already checked if something went wrong so we are at least up to initializing.
		// Could be further along but later components will sort that out.
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Status = summonv1beta1.StatusInitializing
			return setPostgresStatus(obj)
		}
	}
	// If we got this far, we must in ClusterStatusUnknown, who even knows.
	return res, nil
}

func (comp *postgresComponent) reconcileOperatorDatabase(ctx *components.ComponentContext, instance *summonv1beta1.SummonPlatform) (components.Result, *postgresv1.Postgresql, error) {
	res, _, err := ctx.CreateOrUpdate(comp.operatorTemplatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.PostgresOperatorDatabase)
		existing := existingObj.(*dbv1beta1.PostgresOperatorDatabase)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, nil, errors.Wrapf(err, "postgres: failed to create or update postgresoperatordatabase object")
	}

	fetchPostgres := &postgresv1.Postgresql{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Spec.Database.SharedDatabaseName + "-database", Namespace: instance.Namespace}, fetchPostgres)
	if err != nil {
		return res, nil, errors.Wrapf(err, "postgres: failed to get shared database object")
	}
	return res, fetchPostgres, nil
}

func (comp *postgresComponent) reconcileExclusiveDatabase(ctx *components.ComponentContext, instance *summonv1beta1.SummonPlatform) (components.Result, *postgresv1.Postgresql, error) {
	var existingDatabase *postgresv1.Postgresql
	res, _, err := ctx.CreateOrUpdate(comp.postgresTemplatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*postgresv1.Postgresql)
		existingDatabase = existingObj.(*postgresv1.Postgresql)
		// Copy the Spec over.
		existingDatabase.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return res, existingDatabase, errors.Wrap(err, "postgres: error with create or update of exclusive database")
	}
	return res, existingDatabase, nil
}
