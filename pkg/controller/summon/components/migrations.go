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

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type migrationComponent struct {
	templatePath string
}

func NewMigrations(templatePath string) *migrationComponent {
	return &migrationComponent{templatePath: templatePath}
}

func (comp *migrationComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.Migration{},
	}
}

func (_ *migrationComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	if instance.Status.PostgresStatus != dbv1beta1.StatusReady {
		// Database not ready yet.
		return false
	}
	if instance.Status.PullSecretStatus != secretsv1beta1.StatusReady {
		// Pull secret not ready yet.
		return false
	}
	return true
}

func (comp *migrationComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Originally a check done in IsReconcilable, but because of autodeploy setting Spec.Version during
	// Reconcile stage, check has to be done here to see if Spec.Version value was set by autodeploy.
	if instance.Status.BackupVersion != instance.Spec.Version {
		return components.Result{}, nil
	}

	if instance.Spec.Version == instance.Status.MigrateVersion {
		// Already migrated, update status and move on.
		return components.Result{StatusModifier: setStatus(summonv1beta1.StatusDeploying)}, nil
	}

	var existing *dbv1beta1.Migration
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.Migration)
		existing = existingObj.(*dbv1beta1.Migration)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		// Doing check here instead of template purely for ease of use
		existing.Spec.NoCore1540Fixup = instance.Spec.NoCore1540Fixup || instance.Status.MigrateVersion == ""
		return nil
	})
	if err != nil {
		return res, err
	}

	if existing.Status.Status == dbv1beta1.StatusReady {
		// Once status is ready delete it.
		err = comp.deleteMigration(ctx, existing)
		if err != nil {
			return components.Result{}, err
		}
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Status = summonv1beta1.StatusPostMigrateWait
			instance.Status.MigrateVersion = existing.Spec.Version
			return nil
		}}, nil
	}

	// If the job failed pass the error straight through into summon controller
	if existing.Status.Status == dbv1beta1.StatusError {
		fmt.Println("Executing error block.")
		return components.Result{}, errors.New(existing.Status.Message)
	}

	// Job is still running, will get reconciled when it finishes.
	return components.Result{StatusModifier: setStatus(summonv1beta1.StatusMigrating)}, nil
}

func (_ *migrationComponent) deleteMigration(ctx *components.ComponentContext, target *dbv1beta1.Migration) error {
	err := ctx.Delete(ctx.Context, target, nil)
	if err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrap(err, "migrations: failed to delete migration object")
	}
	return nil
}
