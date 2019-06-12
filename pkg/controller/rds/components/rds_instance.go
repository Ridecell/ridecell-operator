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
	"strings"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	corev1 "k8s.io/api/core/v1"
)

const RDSInstanceDatabaseFinalizer = "rdsinstance.database.finalizer"

type rdsInstanceComponent struct {
	rdsAPI rdsiface.RDSAPI
}

func NewRDSInstance() *rdsInstanceComponent {
	sess := session.Must(session.NewSession())
	rdsService := rds.New(sess)
	return &rdsInstanceComponent{rdsAPI: rdsService}
}

func (comp *rdsInstanceComponent) InjectRDSAPI(rdsapi rdsiface.RDSAPI) {
	comp.rdsAPI = rdsapi
}

func (_ *rdsInstanceComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *rdsInstanceComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	return true
}

func (comp *rdsInstanceComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		if !helpers.ContainsFinalizer(RDSInstanceDatabaseFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(RDSInstanceDatabaseFinalizer, instance)
			err := ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to update instance while adding finalizer")
			}
		}
	} else {
		if helpers.ContainsFinalizer(RDSInstanceDatabaseFinalizer, instance) {
			describeDBInstancesOutput, err := comp.rdsAPI.DescribeDBInstances(&rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(instance.Spec.InstanceID),
			})
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok && aerr.Code() != rds.ErrCodeDBInstanceNotFoundFault {
					return components.Result{}, errors.Wrapf(err, "rds: unable to describe db instance")
				}
			} else {
				// If there was no error our instance exists
				if aws.StringValue(describeDBInstancesOutput.DBInstances[0].DBInstanceStatus) == "deleting" {
					return components.Result{RequeueAfter: time.Minute * 1}, nil
				}
				// if the instance is not currently being deleted, attempt a delete and exit accordingly.
				result, err := comp.deleteDependencies(ctx)
				return result, err
			}

			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(RDSInstanceDatabaseFinalizer, instance)
			err = ctx.Update(ctx.Context, instance.DeepCopy())
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "rds: failed to update instance while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// Get our password secret
	fetchSecret := &corev1.Secret{}
	err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.rds-user-password", instance.Name), Namespace: instance.Namespace}, fetchSecret)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to get password secret")
	}

	password, ok := fetchSecret.Data["password"]
	if !ok {
		return components.Result{}, errors.New("rds: database password secret not found")
	}

	var databaseNotExist bool
	var database *rds.DBInstance
	databaseUsername := strings.Replace(instance.Spec.Username, "-", "_", -1)

	if instance.Spec.SubnetGroupName == "" {
		return components.Result{}, errors.New("rds: aws_subnet_group_name var not set")
	}

	describeDBInstancesOutput, err := comp.rdsAPI.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instance.Spec.InstanceID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != rds.ErrCodeDBInstanceNotFoundFault {
			return components.Result{}, errors.Wrapf(err, "rds: unable to describe db instance")
		}
		databaseNotExist = true
	}

	if databaseNotExist {
		createDBInstanceOutput, err := comp.rdsAPI.CreateDBInstance(&rds.CreateDBInstanceInput{
			MasterUsername:             aws.String(databaseUsername),
			DBInstanceIdentifier:       aws.String(instance.Spec.InstanceID),
			MasterUserPassword:         aws.String(string(password)),
			StorageType:                aws.String("gp2"),
			AllocatedStorage:           aws.Int64(instance.Spec.AllocatedStorage),
			DBInstanceClass:            aws.String(instance.Spec.InstanceClass),
			PreferredMaintenanceWindow: aws.String(instance.Spec.MaintenanceWindow),
			Engine:                     aws.String(instance.Spec.Engine),
			EngineVersion:              aws.String(instance.Spec.EngineVersion),
			MultiAZ:                    instance.Spec.MultiAZ,
			PubliclyAccessible:         aws.Bool(true),
			DBParameterGroupName:       aws.String(instance.Name),
			VpcSecurityGroupIds:        []*string{aws.String(instance.Status.SecurityGroupID)},
			DBSubnetGroupName:          aws.String(instance.Spec.SubnetGroupName),
			StorageEncrypted:           aws.Bool(true),
			Tags: []*rds.Tag{
				&rds.Tag{
					Key:   aws.String("Ridecell-Operator"),
					Value: aws.String("true"),
				},
				&rds.Tag{
					Key:   aws.String("tenant"),
					Value: aws.String(instance.Name),
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "rds: unable to create db instance")
		}
		database = createDBInstanceOutput.DBInstance
	} else {
		database = describeDBInstancesOutput.DBInstances[0]
	}

	// Handle tagging
	listTagsForResourceOutput, err := comp.rdsAPI.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: database.DBInstanceArn,
	})
	if err != nil {
		return components.Result{}, errors.Wrap(err, "rds: failed to list database tags")
	}
	var foundOperatorTag bool
	var foundTenantTag bool
	for _, tag := range listTagsForResourceOutput.TagList {
		if aws.StringValue(tag.Key) == "Ridecell-Operator" && aws.StringValue(tag.Value) == "true" {
			foundOperatorTag = true
		}
		if aws.StringValue(tag.Key) == "tenant" && aws.StringValue(tag.Value) == instance.Name {
			foundTenantTag = true
		}
	}

	var tagsToAdd []*rds.Tag
	if !foundOperatorTag {
		tagsToAdd = append(tagsToAdd, &rds.Tag{Key: aws.String("Ridecell-Operator"), Value: aws.String("true")})
	}
	if !foundTenantTag {
		tagsToAdd = append(tagsToAdd, &rds.Tag{Key: aws.String("tentant"), Value: aws.String(instance.Name)})
	}
	if len(tagsToAdd) > 0 {
		_, err = comp.rdsAPI.AddTagsToResource(&rds.AddTagsToResourceInput{
			ResourceName: database.DBInstanceArn,
			Tags:         tagsToAdd,
		})
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to add tags to database")
		}
	}

	var needsUpdate bool
	// For now we're only making changes that are safe to apply immediately
	// This does exclude instance size for now
	databaseModifyInput := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: database.DBInstanceIdentifier,
		ApplyImmediately:     aws.Bool(true),
	}
	// TODO: Things could get weird if allocated storage is increased by less than 10% as aws will automatically round up to the nearest 10% increase
	// This is pretty unlikely to happen even at larger numbers.
	if aws.Int64Value(database.AllocatedStorage) != instance.Spec.AllocatedStorage {
		needsUpdate = true
		databaseModifyInput.AllocatedStorage = aws.Int64(instance.Spec.AllocatedStorage)
	}

	// attempt a database query to test see if our password is correct.
	// only attempt this when database is in ready state.
	if instance.Status.Status == dbv1beta1.StatusReady {
		db, err := postgres.Open(ctx, &instance.Status.Connection)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to open db connection")
		}
		databaseRows, err := db.Query(`SELECT 1;`)
		if err != nil {
			// If the error is invalid password update the database to reflect the expected password
			// 28P01 == Invalid Password
			if pqerr, ok := err.(*pq.Error); ok && pqerr.Code == "28P01" {
				databaseModifyInput.MasterUserPassword = aws.String(string(password))
				needsUpdate = true
			} else {
				return components.Result{}, errors.Wrap(err, "rds: failed to query database")
			}
		} else {
			defer databaseRows.Close()
		}
	}

	dbStatus := aws.StringValue(database.DBInstanceStatus)
	// Only try to update the database if the status is available, otherwise a change may already be in progress.
	if (dbStatus == "available" || dbStatus == "pending-reboot") && needsUpdate {
		err = comp.modifyRDSInstance(databaseModifyInput)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "rds: failed to modify db instance")
		}
		return components.Result{RequeueAfter: time.Second * 30}, nil
	}

	if dbStatus == "failed" || dbStatus == "incompatible-parameters" {
		return components.Result{}, errors.New("rds: rds instance is in a failure state")
	}

	if dbStatus == "modifying" || dbStatus == "resetting-master-credentials" || dbStatus == "backing-up" {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.RDSInstance)
			instance.Status.Status = dbv1beta1.StatusModifying
			instance.Status.Message = "password is being updated"
			return nil
		}, RequeueAfter: time.Second * 30}, nil
	}

	if dbStatus == "creating" {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.RDSInstance)
			instance.Status.InstanceID = aws.StringValue(database.DBInstanceIdentifier)
			instance.Status.Status = dbv1beta1.StatusCreating
			instance.Status.Message = fmt.Sprintf("RDS instance status: %s", dbStatus)
			return nil
		}, RequeueAfter: time.Second * 30}, nil
	}

	if dbStatus == "available" || dbStatus == "pending-reboot" {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.RDSInstance)
			instance.Status.Status = dbv1beta1.StatusReady
			instance.Status.Message = "RDS instance exists and is available"
			instance.Status.InstanceID = aws.StringValue(database.DBInstanceIdentifier)
			instance.Status.Connection.Host = aws.StringValue(database.Endpoint.Address)
			instance.Status.Connection.Port = 5432
			instance.Status.Connection.Username = aws.StringValue(database.MasterUsername)
			instance.Status.Connection.Database = "postgres"
			return nil
		}}, nil
	}

	// catchall for i have no idea why this happened, retry every minute just in case it's weird
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RDSInstance)
		instance.Status.Status = dbv1beta1.StatusUnknown
		instance.Status.Message = fmt.Sprintf("RDS instance is in an unknown or unhandled state: %s", dbStatus)
		return nil
	}, RequeueAfter: time.Second * 30}, nil
}

func (comp *rdsInstanceComponent) modifyRDSInstance(modifyInput *rds.ModifyDBInstanceInput) error {
	_, err := comp.rdsAPI.ModifyDBInstance(modifyInput)
	if err != nil {
		return errors.Wrap(err, "rds: failed to update rds instance")
	}
	return nil
}

func (comp *rdsInstanceComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)

	_, err := comp.rdsAPI.DeleteDBInstance(&rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:      aws.String(instance.Spec.InstanceID),
		FinalDBSnapshotIdentifier: aws.String(fmt.Sprintf("%s-%s", instance.Spec.InstanceID, time.Now().UTC().Format("2006-01-02-15-04"))),
	})

	if err != nil {
		// This obnoxious block of error checking reduces api calls and error spam.
		aerr, ok := err.(awserr.Error)
		if ok {
			if aerr.Code() != rds.ErrCodeDBInstanceNotFoundFault {
				// If the instance isn't ready to be deleted wait a minute and try again
				if aerr.Code() == rds.ErrCodeInvalidDBInstanceStateFault {
					return components.Result{RequeueAfter: time.Minute * 1}, nil
				}
				return components.Result{}, errors.Wrap(err, "rds: failed to delete db for finalizer")
			}
		} else {
			return components.Result{}, errors.Wrap(err, "rds: failed to delete db for finalizer")
		}
	}
	return components.Result{Requeue: true}, nil
}
