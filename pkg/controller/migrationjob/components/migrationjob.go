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

	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

const MigrationJobFinalizer = "migration.finalizer"

type migrationJobComponent struct {
}

func NewMigrationJob() *migrationJobComponent {
	return &migrationJobComponent{}
}

func (comp *migrationJobComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&batchv1.Job{},
	}
}

func (_ *migrationJobComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *migrationJobComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.MigrationJob)

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(MigrationJobFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(MigrationJobFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "migration: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(MigrationJobFinalizer, instance) {
			result, err := comp.deleteDependencies(ctx)
			if err != nil {
				return result, err
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(MigrationJobFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "migration: failed to update object while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// If status is ready we've already migrated. Object will be cleaned up by summon controller.
	if instance.Status.Status == dbv1beta1.StatusReady {
		return components.Result{}, nil
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-migrations", instance.Name),
			Namespace: instance.Namespace,
		},
		Spec: instance.Spec,
	}

	existing := &batchv1.Job{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, existing)
	if err != nil && kerrors.IsNotFound(err) {
		glog.Infof("Creating migration Job %s/%s\n", job.Namespace, job.Name)
		err = controllerutil.SetControllerReference(instance, job, ctx.Scheme)
		if err != nil {
			return components.Result{}, err
		}

		err = ctx.Create(ctx.Context, job)
		if err != nil {
			// If this fails, someone else might have started a migraton job between the Get and here, so just try again.
			return components.Result{}, errors.Wrapf(err, "migrations: error creation migration job %s/%s, might have lost the race condition", job.Namespace, job.Name)
		}
		// Job is started, so we're done for now
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.MigrationJob)
			instance.Status.Status = dbv1beta1.StatusMigrating
			return nil
		}}, nil
	} else if err != nil {
		// Some other real error, bail.
		return components.Result{}, err
	}

	// If we get this far, the job previously started at some point and might be done.
	// First make sure we even care about this job, it only counts if it's for the version we want.
	newVersion, ok := instance.Labels["app.kubernetes.io/version"]
	if !ok {
		return components.Result{}, errors.Wrapf(err, "migrations: unable to find version label")
	}
	existingVersion, ok := existing.Spec.Template.Labels["app.kubernetes.io/version"]
	if !ok || existingVersion != newVersion {
		glog.Infof("[%s/%s] migrations: Found existing migration job with bad version %#v\n", instance.Namespace, instance.Name, existingVersion)
		// This is from a bad (or broken if !ok) version, try to delete it and then run again.
		err = ctx.Delete(ctx.Context, existing, client.PropagationPolicy(metav1.DeletePropagationBackground))
		// Pass in redundant requeue as err is sometimes nil here
		return components.Result{Requeue: true}, errors.Wrapf(err, "migrations: found existing migration job %s/%s with bad version %#v", instance.Namespace, instance.Name, existingVersion)
	}

	// Check if the job succeeded.
	if existing.Status.Succeeded > 0 {
		// Success! Update the MigrateVersion (this will trigger a reconcile) and delete the job.
		glog.V(2).Infof("[%s/%s] Deleting migration Job %s/%s\n", instance.Namespace, instance.Name, existing.Namespace, existing.Name)
		err = ctx.Delete(ctx.Context, existing, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "migrations: error deleting successful migration job %s/%s", existing.Namespace, existing.Name)
		}

		glog.Infof("[%s/%s] migrations: Migration job succeeded for version %s\n", instance.Namespace, instance.Name, newVersion)

		// Onward to deploying!
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.MigrationJob)
			instance.Status.Status = dbv1beta1.StatusReady
			return nil
		}}, nil
	}

	// ... Or if the job failed.
	if existing.Status.Failed > 0 {
		// If it was an outdated job, we would have already deleted it, so this means it's a failed migration for the current version.
		glog.Errorf("[%s/%s] Migration job failed, leaving job %s/%s for debugging purposes\n", instance.Namespace, instance.Name, existing.Namespace, existing.Name)
		return components.Result{}, errors.Errorf("migrations: migration job %s/%s failed", existing.Namespace, existing.Name)
	}

	// Job is still running, will get reconciled when it finishes.
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.MigrationJob)
		instance.Status.Status = dbv1beta1.StatusMigrating
		return nil
	}}, nil
}

func (_ *migrationJobComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.MigrationJob)
	targetJob := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-migrations", instance.Name), Namespace: instance.Namespace}}
	err := ctx.Delete(ctx.Context, targetJob, client.PropagationPolicy(metav1.DeletePropagationBackground))
	if err != nil && !kerrors.IsNotFound(err) {
		return components.Result{}, errors.Wrapf(err, "migrations: error deleting migration job %s/%s", targetJob.Namespace, targetJob.Name)
	}
	return components.Result{}, nil
}
