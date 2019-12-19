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
	"os"
	"strings"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	es "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	esiface "github.com/aws/aws-sdk-go/service/elasticsearchservice/elasticsearchserviceiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	corev1 "k8s.io/api/core/v1"
)

const elasticsearchFinalizer = "elasticsearch.finalizer"

type elasticSearchComponent struct {
	esAPI  esiface.ElasticsearchServiceAPI
	iamAPI iamiface.IAMAPI
}

func NewElasticSearch() *elasticSearchComponent {
	sess := session.Must(session.NewSession())
	esService := es.New(sess)
	iamService := iam.New(sess)
	return &elasticSearchComponent{esAPI: esService, iamAPI: iamService}
}

func (comp *elasticSearchComponent) InjectESAPI(esapi esiface.ElasticsearchServiceAPI, iamapi iamiface.IAMAPI) {
	comp.esAPI = esapi
	comp.iamAPI = iamapi
}

func (_ *elasticSearchComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *elasticSearchComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *elasticSearchComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.ElasticSearch)
	var esDomainInstance *es.ElasticsearchDomainStatus
	esDomainName := strings.ToLower(instance.Name)

	// if object is not being deleted
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Is our finalizer attached to the object?
		if !helpers.ContainsFinalizer(elasticsearchFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(elasticsearchFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "elasticsearch: failed to update instance while adding finalizer")
			}
			return components.Result{Requeue: true}, nil
		}
	} else {
		if helpers.ContainsFinalizer(elasticsearchFinalizer, instance) {
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" && os.Getenv("ENABLE_FINALIZERS") == "true" {
				result, err := comp.deleteDependencies(ctx)
				if err != nil {
					return result, err
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(elasticsearchFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "elasticsearch: failed to update instance while removing finalizer")
			}
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// Wait for security group component to complete
	if instance.Spec.SecurityGroupId == "" {
		return components.Result{Requeue: true}, nil
	}
	//Create Service Role for ElasticSearch
	_, err := comp.iamAPI.CreateServiceLinkedRole(&iam.CreateServiceLinkedRoleInput{
		AWSServiceName: aws.String("es.amazonaws.com"),
		Description:    aws.String("created through ridecell-operator"),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != iam.ErrCodeInvalidInputException {
			return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to create service role for elasticsearch")
		}
	}

	var elasticsearchNotExist bool
	// try to get ES instance
	describeElasticsearchDomainOutput, err := comp.esAPI.DescribeElasticsearchDomain(&es.DescribeElasticsearchDomainInput{DomainName: aws.String(esDomainName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() != es.ErrCodeResourceNotFoundException {
			return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to describe elasticsearch instance")
		}
		elasticsearchNotExist = true
	}

	if elasticsearchNotExist {
		// Domain access policy: It will allow all connections within the current VPC
		accessPolicy := fmt.Sprintf("{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":{\"AWS\":\"*\"},\"Action\":\"es:*\",\"Resource\":\"arn:aws:es:us-west-2:439671274615:domain/%s/*\"}]}", esDomainName)

		// ElasticSearch Cluster config
		esClusterConfig := &es.ElasticsearchClusterConfig{
			DedicatedMasterEnabled: aws.Bool(false),
			InstanceCount:          aws.Int64(instance.Spec.NoOfInstances),
			InstanceType:           aws.String(instance.Spec.InstanceType),
		}
		vpcOptions := &es.VPCOptions{
			SecurityGroupIds: aws.StringSlice([]string{instance.Spec.SecurityGroupId}),
			SubnetIds:        aws.StringSlice([]string{instance.Spec.SubnetIds[0]}),
		}

		// Modify ES config accroding to deployment type
		if instance.Spec.DeploymentType == "Production" {
			vpcOptions.SubnetIds = aws.StringSlice(instance.Spec.SubnetIds)
			esClusterConfig.DedicatedMasterEnabled = aws.Bool(true)
			esClusterConfig.DedicatedMasterType = aws.String(instance.Spec.InstanceType)
			esClusterConfig.ZoneAwarenessEnabled = aws.Bool(true)
			esClusterConfig.ZoneAwarenessConfig = &es.ZoneAwarenessConfig{
				AvailabilityZoneCount: aws.Int64(int64(len(instance.Spec.SubnetIds))),
			}
			//esClusterConfig.DedicatedMasterCount = aws.Int64(3) // By default, the count is 3
		}

		// Create ES domain with given configs
		createElasticsearchDomainOutput, err := comp.esAPI.CreateElasticsearchDomain(&es.CreateElasticsearchDomainInput{
			DomainName: aws.String(esDomainName),
			DomainEndpointOptions: &es.DomainEndpointOptions{
				EnforceHTTPS:      aws.Bool(true),
				TLSSecurityPolicy: aws.String("Policy-Min-TLS-1-2-2019-07"),
			},
			EBSOptions: &es.EBSOptions{
				EBSEnabled: aws.Bool(true),
				VolumeSize: aws.Int64(instance.Spec.StoragePerNode),
			},
			ElasticsearchVersion:       aws.String(instance.Spec.ElasticSearchVersion),
			ElasticsearchClusterConfig: esClusterConfig,
			EncryptionAtRestOptions: &es.EncryptionAtRestOptions{
				Enabled: aws.Bool(true),
			},
			NodeToNodeEncryptionOptions: &es.NodeToNodeEncryptionOptions{
				Enabled: aws.Bool(true),
			},
			//SnapshotOptions: &es.SnapshotOptions{}, will be configured later
			VPCOptions:     vpcOptions,
			AccessPolicies: aws.String(accessPolicy),
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to create elasticsearch instance")
		}
		esDomainInstance = createElasticsearchDomainOutput.DomainStatus
	} else {
		esDomainInstance = describeElasticsearchDomainOutput.DomainStatus
	}

	if aws.BoolValue(esDomainInstance.Processing) || aws.BoolValue(esDomainInstance.UpgradeProcessing) {
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*awsv1beta1.ElasticSearch)
			instance.Status.Status = "Processing"
			instance.Status.Message = "The ElasticSearch domain is processing changes"
			instance.Status.DomainEndpoint = aws.StringValue(esDomainInstance.Endpoints["vpc"])
			return nil
		}, RequeueAfter: time.Second * 60}, nil
	}

	// Set Ridecell-Operator tag if not present
	listTagsOuput, err := comp.esAPI.ListTags(&es.ListTagsInput{
		ARN: esDomainInstance.ARN,
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to get tags of elasticsearch instance")
	}

	tagFound := false
	for _, tag := range listTagsOuput.TagList {
		if aws.StringValue(tag.Key) == "Ridecell-Operator" {
			tagFound = true
		}
	}
	if !tagFound {
		_, err := comp.esAPI.AddTags(&es.AddTagsInput{
			ARN: esDomainInstance.ARN,
			TagList: []*es.Tag{
				&es.Tag{Key: aws.String("Ridecell-Operator"), Value: aws.String("true")},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to set tags of elasticsearch instance")
		}
	}

	//Check if ElasticSearch domain needs to be updated
	needsUpdate := false
	updateElasticsearchDomainConfigInput := &es.UpdateElasticsearchDomainConfigInput{
		DomainName:                 aws.String(esDomainName),
		ElasticsearchClusterConfig: &es.ElasticsearchClusterConfig{},
		VPCOptions:                 &es.VPCOptions{},
	}
	// check for no of instances
	if instance.Spec.NoOfInstances != aws.Int64Value(esDomainInstance.ElasticsearchClusterConfig.InstanceCount) {
		updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.InstanceCount = aws.Int64(instance.Spec.NoOfInstances)
		needsUpdate = true
	}
	// check for instance types
	if instance.Spec.InstanceType != aws.StringValue(esDomainInstance.ElasticsearchClusterConfig.InstanceType) {
		updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.InstanceType = aws.String(instance.Spec.InstanceType)
		if aws.BoolValue(esDomainInstance.ElasticsearchClusterConfig.DedicatedMasterEnabled) {
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.DedicatedMasterType = aws.String(instance.Spec.InstanceType)
		}
		needsUpdate = true
	}
	// Deployment type changes
	dedicatedMater := false
	if instance.Spec.DeploymentType == "Production" {
		dedicatedMater = true
	}
	if dedicatedMater != aws.BoolValue(esDomainInstance.ElasticsearchClusterConfig.DedicatedMasterEnabled) {
		if dedicatedMater {
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.DedicatedMasterEnabled = aws.Bool(true)
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.DedicatedMasterType = aws.String(instance.Spec.InstanceType)
			updateElasticsearchDomainConfigInput.VPCOptions.SubnetIds = aws.StringSlice(instance.Spec.SubnetIds)
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.ZoneAwarenessEnabled = aws.Bool(true)
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.ZoneAwarenessConfig = &es.ZoneAwarenessConfig{
				AvailabilityZoneCount: aws.Int64(int64(len(instance.Spec.SubnetIds))),
			}
			//updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.DedicatedMasterCount = aws.Int64(3)
		} else {
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.DedicatedMasterEnabled = aws.Bool(false)
			updateElasticsearchDomainConfigInput.ElasticsearchClusterConfig.ZoneAwarenessEnabled = aws.Bool(false)
			updateElasticsearchDomainConfigInput.VPCOptions.SubnetIds = aws.StringSlice([]string{instance.Spec.SubnetIds[0]})
		}
		needsUpdate = true
	}
	// check for storage per node
	if instance.Spec.StoragePerNode != aws.Int64Value(esDomainInstance.EBSOptions.VolumeSize) {
		updateElasticsearchDomainConfigInput.EBSOptions = &es.EBSOptions{
			VolumeSize: aws.Int64(instance.Spec.StoragePerNode),
		}
		needsUpdate = true
	}

	if needsUpdate {
		// update ES Domain
		_, err := comp.esAPI.UpdateElasticsearchDomainConfig(updateElasticsearchDomainConfigInput)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "elasticsearch: unable to update elasticsearch instance")
		}
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*awsv1beta1.ElasticSearch)
			instance.Status.Status = "Processing"
			instance.Status.Message = "The ElasticSearch domain is processing changes"
			instance.Status.DomainEndpoint = aws.StringValue(esDomainInstance.Endpoints["vpc"])
			return nil
		}, RequeueAfter: time.Second * 60}, nil
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*awsv1beta1.ElasticSearch)
		instance.Status.Status = "Ready"
		instance.Status.Message = "The ElasticSearch domain is ready"
		instance.Status.DomainEndpoint = aws.StringValue(esDomainInstance.Endpoints["vpc"])
		return nil
	}}, nil
}

func (comp *elasticSearchComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.ElasticSearch)

	_, err := comp.esAPI.DeleteElasticsearchDomain(&es.DeleteElasticsearchDomainInput{
		DomainName: aws.String(strings.ToLower(instance.Name)),
	})
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok {
			if aerr.Code() != es.ErrCodeResourceNotFoundException {
				return components.Result{}, errors.Wrap(err, "elasticsearch: failed to delete elasticsearch domain for finalizer")
			}
		}
	}
	return components.Result{}, nil
}
