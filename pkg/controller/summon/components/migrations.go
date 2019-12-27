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
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const flavorBucket = "ridecell-flavors"

type migrationComponent struct {
	templatePath string
}

func NewMigrations(templatePath string) *migrationComponent {
	return &migrationComponent{templatePath: templatePath}
}

func (comp *migrationComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&batchv1.Job{},
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
	// Reconcile stage, check has to be done here since it wouldn't be visible during IsReconcilable stages.
	if instance.Status.BackupVersion != instance.Spec.Version {
		return components.Result{}, nil
	}

	if instance.Spec.Version == instance.Status.MigrateVersion {
		// Already migrated, update status and move on.
		return components.Result{StatusModifier: setStatus(summonv1beta1.StatusDeploying)}, nil
	}

	var urlStr string
	if instance.Spec.Flavor != "" {
		svc := s3.New(session.Must(session.NewSession(&aws.Config{
			Region: aws.String("us-west-2"),
		})))
		req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(flavorBucket),
			Key:    aws.String(fmt.Sprintf("%s.json.bz2", instance.Spec.Flavor)),
		})

		var err error
		urlStr, err = req.Presign(15 * time.Minute)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "migrations: failed to presign s3 url")
		}
	}

	extra := map[string]interface{}{}
	extra["presignedUrl"] = urlStr

	obj, err := ctx.GetTemplate(comp.templatePath, extra)
	if err != nil {
		return components.Result{}, err
	}
	job := obj.(*batchv1.Job)

	existing := &batchv1.Job{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, existing)
	if err != nil && kerrors.IsNotFound(err) {
		glog.Infof("Creating migration Job %s/%s\n", job.Namespace, job.Name)
		err = controllerutil.SetControllerReference(instance, job, ctx.Scheme)
		if err != nil {
			return components.Result{}, err
		}

		err = ctx.Create(ctx.Context, job)
		if err != nil {
			// If this fails, someone else might have started a migraton job between the Get and here, so just try again.
			return components.Result{Requeue: true}, errors.Wrapf(err, "migrations: error creation migration job %s/%s, might have lost the race condition", job.Namespace, job.Name)
		}
		// Job is started, so we're done for now.
		fmt.Printf("DEBUG: Migrations: job has started (%v)...\n", job)
		return components.Result{StatusModifier: setStatus(summonv1beta1.StatusMigrating)}, nil
	} else if err != nil {
		// Some other real error, bail.
		fmt.Printf("DEBUG: Migrations: error in creating job: %v\n", err)
		return components.Result{}, err
	}

	// If we get this far, the job previously started at some point and might be done.
	// First make sure we even care about this job, it only counts if it's for the version we want.
	existingVersion, ok := existing.Labels["app.kubernetes.io/version"]
	if !ok || existingVersion != instance.Spec.Version {
		glog.Infof("[%s/%s] migrations: Found existing migration job with bad version %#v\n", instance.Namespace, instance.Name, existingVersion)
		fmt.Printf("DEBUG: Migrations: Found existing job with bad version %#v...\n", existingVersion)
		// This is from a bad (or broken if !ok) version, try to delete it and then run again.
		err = ctx.Delete(ctx.Context, existing, client.PropagationPolicy(metav1.DeletePropagationBackground))
		return components.Result{Requeue: true}, errors.Wrapf(err, "migrations: found existing migration job %s/%s with bad version %#v", instance.Namespace, instance.Name, existingVersion)
	}

	// Check if the job succeeded.
	if existing.Status.Succeeded > 0 {
		fmt.Printf("DEBUG: Migrations: job succeeded!\n")
		// Success! Update the MigrateVersion (this will trigger a reconcile) and delete the job.
		glog.V(2).Infof("[%s/%s] Deleting migration Job %s/%s\n", instance.Namespace, instance.Name, existing.Namespace, existing.Name)
		fmt.Printf("DEBUG: Migrations: delete successful job %v\n", existing)
		err = ctx.Delete(ctx.Context, existing, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			return components.Result{Requeue: true}, errors.Wrapf(err, "migrations: error deleting successful migration job %s/%s", existing.Namespace, existing.Name)
		}

		glog.Infof("[%s/%s] migrations: Migration job succeeded, updating MigrateVersion from %s to %s\n", instance.Namespace, instance.Name, instance.Status.MigrateVersion, instance.Spec.Version)
		fmt.Printf("DEBUG: Migrations: Updating MigrateVersion to %s...\n", instance.Spec.Version)
		// Store migrate version in the closure to avoid concurrent edits to Spec.Version resulting in incorrectly advancing MigrateVersion.
		migrateVersion := instance.Spec.Version
		// Onward to deploying!
		fmt.Printf("DEBUG: Migrations: Moving to PostMigrateWait...\n")
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Status = summonv1beta1.StatusPostMigrateWait
			instance.Status.MigrateVersion = migrateVersion
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
	return components.Result{StatusModifier: setStatus(summonv1beta1.StatusMigrating)}, nil
}
