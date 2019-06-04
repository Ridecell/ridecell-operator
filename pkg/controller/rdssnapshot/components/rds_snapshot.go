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
	"fmt"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// k8s timestamps aren't friendly with ebs snapshots
const CustomTimeLayout = "2006-01-02-15-04-05"
const RDSSnapshotFinalizer = "rdssnapshot.finalizer"

type RDSSnapshotComponent struct {
	rdsAPI rdsiface.RDSAPI
}

func NewRDSSnapshot() *RDSSnapshotComponent {
	sess := session.Must(session.NewSession())
	rdsService := rds.New(sess)
	return &RDSSnapshotComponent{rdsAPI: rdsService}
}

func (comp *RDSSnapshotComponent) InjectRDSAPI(rdsapi rdsiface.RDSAPI) {
	comp.rdsAPI = rdsapi
}

func (_ *RDSSnapshotComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *RDSSnapshotComponent) WatchChannel() chan event.GenericEvent {
	genericChannel := make(chan event.GenericEvent)
	return genericChannel
}

func (_ *RDSSnapshotComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *RDSSnapshotComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSSnapshot)

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(RDSSnapshotFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(RDSSnapshotFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(RDSSnapshotFinalizer, instance) {
			result, err := comp.deleteDependencies(ctx)
			if err != nil {
				return result, err
			}

			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(RDSSnapshotFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to update object while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	var scheduledDelete bool
	var deletionTime time.Time
	var deletionTimestamp string
	if instance.Spec.TTL != 0 {
		scheduledDelete = true
		deletionTime = instance.ObjectMeta.CreationTimestamp.Add(instance.Spec.TTL)
		deletionTimestamp = time.Time.Format(deletionTime, CustomTimeLayout)

		// Check if our object needs to be cleaned up
		if metav1.Now().After(deletionTime) {
			currentTime := metav1.Now()
			instance.ObjectMeta.SetDeletionTimestamp(&currentTime)
			err := ctx.Client.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrap(err, "rds_snapshot: failed to delete itself")
			}
			return components.Result{Requeue: true}, nil
		}
	}

	snapshotTags := []*rds.Tag{
		&rds.Tag{
			Key:   aws.String("Ridecell-Operator"),
			Value: aws.String("true"),
		},
		&rds.Tag{
			Key:   aws.String("scheduled-for-deletion"),
			Value: aws.String(fmt.Sprintf("%v", scheduledDelete)),
		},
	}
	if scheduledDelete {
		deletionTimestampTag := &rds.Tag{
			Key:   aws.String("deletion-timestamp"),
			Value: aws.String(deletionTimestamp),
		}
		snapshotTags = append(snapshotTags, deletionTimestampTag)
	}

	var dbSnapshot *rds.DBSnapshot
	describeDBSnapshotsOutput, err := comp.rdsAPI.DescribeDBSnapshots(&rds.DescribeDBSnapshotsInput{
		DBSnapshotIdentifier: aws.String(instance.Spec.SnapshotID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != rds.ErrCodeDBSnapshotNotFoundFault {
			return components.Result{}, errors.Wrap(err, "rds_snapshot: failed to describe snapshot")
		}
		// if our snapshot doesn't exist create it
		createDBSnapshotOutput, err := comp.rdsAPI.CreateDBSnapshot(&rds.CreateDBSnapshotInput{
			DBInstanceIdentifier: aws.String(instance.Spec.RDSInstanceID),
			DBSnapshotIdentifier: aws.String(instance.Spec.SnapshotID),
			Tags:                 snapshotTags,
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds_snapshot: failed to create db snapshot")
		}
		dbSnapshot = createDBSnapshotOutput.DBSnapshot

	} else {
		dbSnapshot = describeDBSnapshotsOutput.DBSnapshots[0]
	}

	if aws.StringValue(dbSnapshot.Status) == "pending" {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance.Status.Status = dbv1beta1.StatusCreating
			instance.Status.Message = fmt.Sprintf("Snapshot is in state: %s", aws.StringValue(dbSnapshot.Status))
			return nil
		}, RequeueAfter: time.Minute}, nil
	}

	if aws.StringValue(dbSnapshot.Status) == "error" {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance.Status.Status = dbv1beta1.StatusError
			instance.Status.Message = fmt.Sprintf("Snapshot is in state: %s", aws.StringValue(dbSnapshot.Status))
			return nil
		}}, nil
	}

	// If we got here the snapshot is in "completed" state.
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RDSSnapshot)
		instance.Status.Status = dbv1beta1.StatusReady
		instance.Status.Message = fmt.Sprintf("Snapshot is in state: %s", aws.StringValue(dbSnapshot.Status))
		return nil
	}, RequeueAfter: time.Minute}, nil
}

func (comp *RDSSnapshotComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSSnapshot)

	_, err := comp.rdsAPI.DeleteDBSnapshot(&rds.DeleteDBSnapshotInput{DBSnapshotIdentifier: aws.String(instance.Spec.SnapshotID)})
	if err != nil {
		// if the snapshot isn't found don't consider it an error
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != rds.ErrCodeDBSnapshotNotFoundFault {
			return components.Result{}, errors.Wrap(err, "rds_snapshot: failed to delete rds snapshot")
		}
	}
	return components.Result{}, nil
}
