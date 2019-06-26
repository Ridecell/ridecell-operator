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

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
)

type backupComponent struct {
	templatePath string
}

func NewBackup(templatePath string) *backupComponent {
	return &backupComponent{templatePath: templatePath}
}

func (comp *backupComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&dbv1beta1.RDSSnapshot{},
	}
}

func (_ *backupComponent) IsReconcilable(ctx *components.ComponentContext) bool {
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

func (comp *backupComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Grab PostgresDatabase so we can locate relevant dbconfig
	fetchPostgresDB := &dbv1beta1.PostgresDatabase{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, fetchPostgresDB)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "backup: failed to get postgresdatabase object")
	}

	// Exit early if versions match
	// Exit early if there is no RDS instance
	if instance.Status.BackupVersion == instance.Spec.Version || fetchPostgresDB.Status.RDSInstanceID == "" {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Status = summonv1beta1.StatusDeploying
			instance.Status.BackupVersion = instance.Spec.Version
			return nil
		}}, nil
	}

	backupName := fmt.Sprintf("%s-%s", instance.Name, instance.Spec.Version)
	// Data to be copied over to template
	extra := map[string]interface{}{}
	extra["backupName"] = backupName
	extra["rdsInstanceName"] = fetchPostgresDB.Status.RDSInstanceID

	_, _, err = ctx.CreateOrUpdate(comp.templatePath, extra, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*dbv1beta1.RDSSnapshot)
		existing := existingObj.(*dbv1beta1.RDSSnapshot)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "backup: failed to create or update rds snapshot")
	}

	fetchRDSSnapshot := &dbv1beta1.RDSSnapshot{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: backupName, Namespace: instance.Namespace}, fetchRDSSnapshot)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "backup: failed to get rdssnapshot object")
	}

	if fetchRDSSnapshot.Status.Status == dbv1beta1.StatusError {
		return components.Result{}, errors.Wrapf(err, "backup: rdssnapshot %s is in an error state", fetchRDSSnapshot.Name)
	}

	// We can just return at this point.
	// When the rdssnapshot is finished it will trigger this component to reconcile.
	if fetchRDSSnapshot.Status.Status == dbv1beta1.StatusCreating {
		return components.Result{StatusModifier: setStatus(summonv1beta1.StatusCreatingBackup)}, nil
	}

	// If we got this far our snapshot is ready
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*summonv1beta1.SummonPlatform)
		instance.Status.Status = summonv1beta1.StatusDeploying
		instance.Status.BackupVersion = instance.Spec.Version
		return nil
	}}, nil
}
