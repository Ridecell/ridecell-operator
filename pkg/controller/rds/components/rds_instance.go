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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *rdsInstanceComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*dbv1beta1.RDSInstance)
	if instance.Status.ParameterGroupStatus != dbv1beta1.StatusReady {
		return false
	}
	if instance.Status.SecurityGroupStatus != dbv1beta1.StatusReady {
		return false
	}
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
				DBInstanceIdentifier: aws.String(instance.Name),
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

	var databaseNotExist bool
	var database *rds.DBInstance
	var password string
	databaseName := strings.Replace(instance.Spec.DatabaseName, "-", "_", -1)
	databaseUsername := strings.Replace(instance.Spec.Username, "-", "_", -1)

	if instance.Spec.SubnetGroupName == "" {
		return components.Result{}, errors.New("rds: aws_subnet_group_name var not set")
	}

	describeDBInstancesOutput, err := comp.rdsAPI.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instance.Name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != rds.ErrCodeDBInstanceNotFoundFault {
			return components.Result{}, errors.Wrapf(err, "rds: unable to describe db instance")
		}
		databaseNotExist = true
	}

	if databaseNotExist {
		password = createDBPassword()
		createDBInstanceOutput, err := comp.rdsAPI.CreateDBInstance(&rds.CreateDBInstanceInput{
			MasterUsername:             aws.String(databaseUsername),
			DBInstanceIdentifier:       aws.String(instance.Name),
			DBName:                     aws.String(databaseName),
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
			Tags: []*rds.Tag{
				&rds.Tag{
					Key:   aws.String("ridecell-operator"),
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

	var needsUpdate bool
	// For now we're only making changes that are safe to apply immediately
	// This does exclude instance size for now
	databaseModifyInput := &rds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(instance.Name),
		ApplyImmediately:     aws.Bool(true),
	}
	// TODO: Things could get weird if allocated storage is increased by less than 10% as aws will automatically round up to the nearest 10% increase
	// This is pretty unlikely to happen even at larger numbers.
	if aws.Int64Value(database.AllocatedStorage) != instance.Spec.AllocatedStorage {
		needsUpdate = true
		databaseModifyInput.AllocatedStorage = aws.Int64(instance.Spec.AllocatedStorage)
	}

	// If a new password wasn't created we're going to try to get it from the secret
	// If that secret does not exist we create a new password.
	if len(password) == 0 {
		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.rds.credentials", instance.Name), Namespace: instance.Namespace}, fetchSecret)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return components.Result{}, errors.Wrap(err, "rds: unable to get database secret")
			}
			// Safety case, we don't have the secret but we have the instance, make a new one and update the instance.
			needsUpdate = true
			password = createDBPassword()
			databaseModifyInput.MasterUserPassword = aws.String(password)
		}

		dbPassword, ok := fetchSecret.Data["password"]
		if ok {
			password = string(dbPassword)
		}
		// If the key didn't exist or the fetched password is blank
		if !ok || len(dbPassword) == 0 {
			password = createDBPassword()
			databaseModifyInput.MasterUserPassword = aws.String(password)
		}
	}

	var dbEndpoint string
	if database.Endpoint != nil {
		dbEndpoint = aws.StringValue(database.Endpoint.Address)
	} else {
		dbEndpoint = ""
	}
	secretMap := map[string][]byte{
		"username": []byte(aws.StringValue(database.MasterUsername)),
		"endpoint": []byte(dbEndpoint),
		"password": []byte(password),
	}

	// Deal with master password secret
	dbSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.rds.credentials", instance.Name), Namespace: instance.Namespace}}
	dbSecret.Data = secretMap
	_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, dbSecret.DeepCopy(), func(existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		existing.ObjectMeta.Labels = dbSecret.ObjectMeta.Labels
		existing.ObjectMeta.Annotations = dbSecret.ObjectMeta.Annotations
		existing.Type = dbSecret.Type
		existing.Data = dbSecret.Data
		return nil
	})
	if err != nil {
		return components.Result{}, err
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
			instance.Status.Connection = dbv1beta1.PostgresConnection{
				Host:     aws.StringValue(database.Endpoint.Address),
				Port:     int(5432),
				Username: aws.StringValue(database.MasterUsername),
				Database: databaseName,
				PasswordSecretRef: helpers.SecretRef{
					Name: fmt.Sprintf("%s.rds.credentials", instance.Name),
					Key:  "password",
				},
			}
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
		DBInstanceIdentifier:      aws.String(instance.Name),
		FinalDBSnapshotIdentifier: aws.String(fmt.Sprintf("%s-%s", instance.Name, time.Now().UTC().Format("2006-01-02-15-04"))),
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

func createDBPassword() string {
	// Create a password for our database
	rawPassword := make([]byte, 32)
	rand.Read(rawPassword)
	password := make([]byte, base64.RawURLEncoding.EncodedLen(32))
	base64.RawURLEncoding.Encode(password, rawPassword)
	return string(password)
}
