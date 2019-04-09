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

	var databaseNotExist bool
	var database *rds.DBInstance
	var password []byte
	databaseName := strings.Replace(instance.Name, "-", "_", -1)

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
	} else {
		// if we found the database use it
		database = describeDBInstancesOutput.DBInstances[0]
	}

	if databaseNotExist {
		password = createDBPassword()
		// Create our database
		createDBInstanceOutput, err := comp.rdsAPI.CreateDBInstance(&rds.CreateDBInstanceInput{
			MasterUsername:             aws.String(instance.Spec.Username),
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
	}

	// If we didn't create a new password get the password from the secret.
	if len(password) == 0 {
		fetchSecret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.rds.credentials", instance.Name), Namespace: instance.Namespace}, fetchSecret)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return components.Result{}, errors.Wrap(err, "rds: unable to get database secret")
			}
			// Safety case, we don't have the password but we have the instance,
			// overwrite the password as there's no api call to retrieve it.
			// This is a no downtime update at least on the RDS side.
			password = createDBPassword()
			_, err := comp.rdsAPI.ModifyDBInstance(&rds.ModifyDBInstanceInput{
				DBInstanceIdentifier: database.DBInstanceIdentifier,
				MasterUserPassword:   aws.String(string(password)),
				ApplyImmediately:     aws.Bool(true),
			})
			if err != nil {
				return components.Result{}, errors.Wrap(err, "rds: failed to update rds instance with new password")
			}
		}

		dbPassword, ok := fetchSecret.Data["password"]
		if ok {
			password = dbPassword
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
		"password": password,
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
		return components.Result{Requeue: true}, err
	}

	dbStatus := aws.StringValue(database.DBInstanceStatus)
	if dbStatus != "available" && dbStatus != "pending-reboot" {
		if dbStatus == "failed" || dbStatus == "incompatible-parameters" {
			return components.Result{}, errors.New("rds: rds instance is in a failure state")
		}

		if dbStatus == "modifying" {
			return components.Result{StatusModifier: func(obj runtime.Object) error {
				instance := obj.(*dbv1beta1.RDSInstance)
				instance.Status.Status = dbv1beta1.StatusModifying
				instance.Status.Message = "password is being updated"
				return nil
			}, Requeue: true}, nil
		}
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*dbv1beta1.RDSInstance)
			instance.Status.InstanceID = aws.StringValue(database.DBInstanceIdentifier)
			instance.Status.Status = dbv1beta1.StatusCreating
			instance.Status.Message = fmt.Sprintf("RDS instance status: %s", dbStatus)
			return nil
		}, Requeue: true}, nil
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.RDSInstance)
		instance.Status.Status = dbv1beta1.StatusReady
		instance.Status.Message = "RDS instance exists and is available"
		instance.Status.InstanceID = aws.StringValue(database.DBInstanceIdentifier)
		instance.Status.RDSConnection = dbv1beta1.PostgresConnection{
			Host:     aws.StringValue(database.Endpoint.Address),
			Port:     uint16(5432),
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

func createDBPassword() []byte {
	// Limit this to '/', '@', '"', ' ' | this isn't that big of a deal as it will just loop reconciles
	// Create a password for our new database
	rawPassword := make([]byte, 32)
	rand.Read(rawPassword)
	password := make([]byte, base64.RawStdEncoding.EncodedLen(32))
	base64.RawStdEncoding.Encode(password, rawPassword)
	return password
}
