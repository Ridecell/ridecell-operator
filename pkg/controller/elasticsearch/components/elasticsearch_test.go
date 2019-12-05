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

package components_test

import (
	"os"
	"strings"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	es "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	esiface "github.com/aws/aws-sdk-go/service/elasticsearchservice/elasticsearchserviceiface"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	escomponents "github.com/Ridecell/ridecell-operator/pkg/controller/elasticsearch/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockESClient struct {
	esiface.ElasticsearchServiceAPI
	mockDomainExists  bool
	mockDomainHasTags bool
	mockDomainUpdated bool
	deleteDomain      bool
	finalizerTest     bool
}

var _ = Describe("elasticsearch aws Component", func() {
	os.Setenv("AWS_REGION", "us-west-2")
	comp := escomponents.NewElasticSearch()
	var mockES *mockESClient

	BeforeEach(func() {
		//comp = escomponents.NewDefaults()
		comp = escomponents.NewElasticSearch()
		mockES = &mockESClient{}
		comp.InjectESAPI(mockES)
		// Finalizer is added here to skip the return in reconcile after adding finalizer
		instance.ObjectMeta.Finalizers = []string{"elasticsearch.finalizer"}
		instance.Spec.SubnetIds = append(instance.Spec.SubnetIds, "subnet-12345")
	})

	It("Is Reconcilable", func() {
		Expect(comp.IsReconcilable(ctx)).To(BeTrue())
	})

	It("runs basic reconcile with no existing elasticsearch domain", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal("Processing"))
		Expect(mockES.mockDomainExists).To(BeTrue())
	})

	It("has tags attached", func() {
		instance.Spec.NoOfInstances = 1
		instance.Spec.StoragePerNode = 10
		instance.Spec.DeploymentType = "Development"
		instance.Spec.InstanceType = "r5.large.elasticsearch"
		mockES.mockDomainExists = true
		mockES.mockDomainHasTags = false
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal("Ready"))
		Expect(mockES.mockDomainHasTags).To(BeTrue())
	})

	It("will update the ES domain", func() {
		mockES.mockDomainExists = true
		mockES.mockDomainHasTags = true
		mockES.mockDomainUpdated = false
		instance.Spec.StoragePerNode = 12
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mockES.mockDomainUpdated).To(BeTrue())
	})

	Describe("finalizer tests", func() {
		BeforeEach(func() {
			os.Setenv("ENABLE_FINALIZERS", "true")
		})

		It("adds finalizer when there isn't one", func() {
			mockES.finalizerTest = true
			instance.ObjectMeta.Finalizers = []string{}

			Expect(comp).To(ReconcileContext(ctx))
			fetchESDomain := &awsv1beta1.ElasticSearch{}
			err := ctx.Client.Get(ctx.Context, types.NamespacedName{Name: "test-domain", Namespace: "default"}, fetchESDomain)
			Expect(err).ToNot(HaveOccurred())
			Expect(fetchESDomain.ObjectMeta.Finalizers).To(Equal([]string{"elasticsearch.finalizer"}))
		})

		It("sets deletiontimestamp to non-zero with mock objects existing", func() {
			mockES.finalizerTest = true
			mockES.mockDomainExists = true
			currentTime := metav1.Now()
			instance.ObjectMeta.SetDeletionTimestamp(&currentTime)

			Expect(comp).To(ReconcileContext(ctx))
			Expect(mockES.deleteDomain).To(BeTrue())
		})
	})
})

// Mock aws functions below
func (m *mockESClient) DescribeElasticsearchDomain(input *es.DescribeElasticsearchDomainInput) (*es.DescribeElasticsearchDomainOutput, error) {
	if aws.StringValue(input.DomainName) != strings.ToLower(instance.Name) {
		return nil, errors.New("awsmock_describeESdomain: given domain name does not match spec")
	}
	if !m.mockDomainExists {
		return nil, awserr.New(es.ErrCodeResourceNotFoundException, "awsmock_describeESdomain: ES domain does not exist", errors.New(""))
	}
	return &es.DescribeElasticsearchDomainOutput{
		DomainStatus: &es.ElasticsearchDomainStatus{
			ARN:        aws.String("arn:aws:es:us-west-2:1234567890:domain/test-domain"),
			DomainName: input.DomainName,
			EBSOptions: &es.EBSOptions{
				EBSEnabled: aws.Bool(true),
				VolumeSize: aws.Int64(10),
			},
			ElasticsearchVersion: aws.String("7.1"),
			ElasticsearchClusterConfig: &es.ElasticsearchClusterConfig{
				DedicatedMasterEnabled: aws.Bool(false),
				InstanceCount:          aws.Int64(1),
				InstanceType:           aws.String("r5.large.elasticsearch"),
			},
			Processing:        aws.Bool(false),
			UpgradeProcessing: aws.Bool(false),
		},
	}, nil
}

func (m *mockESClient) CreateElasticsearchDomain(input *es.CreateElasticsearchDomainInput) (*es.CreateElasticsearchDomainOutput, error) {
	m.mockDomainExists = true
	m.mockDomainHasTags = false
	return &es.CreateElasticsearchDomainOutput{
		DomainStatus: &es.ElasticsearchDomainStatus{
			ARN:        aws.String("arn:aws:es:us-west-2:1234567890:domain/test-domain"),
			DomainName: input.DomainName,
			EBSOptions: &es.EBSOptions{
				EBSEnabled: aws.Bool(true),
				VolumeSize: aws.Int64(10),
			},
			ElasticsearchVersion: aws.String("7.1"),
			ElasticsearchClusterConfig: &es.ElasticsearchClusterConfig{
				DedicatedMasterEnabled: aws.Bool(false),
				InstanceCount:          aws.Int64(1),
				InstanceType:           aws.String("r5.large.elasticsearch"),
			},
			Processing:        aws.Bool(true),
			UpgradeProcessing: aws.Bool(false),
		},
	}, nil
}

func (m *mockESClient) UpdateElasticsearchDomainConfig(input *es.UpdateElasticsearchDomainConfigInput) (*es.UpdateElasticsearchDomainConfigOutput, error) {
	m.mockDomainUpdated = true
	return &es.UpdateElasticsearchDomainConfigOutput{}, nil
}

func (m *mockESClient) DeleteElasticsearchDomain(input *es.DeleteElasticsearchDomainInput) (*es.DeleteElasticsearchDomainOutput, error) {
	m.deleteDomain = true
	return &es.DeleteElasticsearchDomainOutput{}, nil
}

func (m *mockESClient) ListTags(input *es.ListTagsInput) (*es.ListTagsOutput, error) {
	if aws.StringValue(input.ARN) != "arn:aws:es:us-west-2:1234567890:domain/test-domain" {
		return nil, awserr.New(es.ErrCodeResourceNotFoundException, "awsmock_listtags: ES domain does not exist", errors.New(""))
	}
	return &es.ListTagsOutput{
		TagList: []*es.Tag{},
	}, nil
}

func (m *mockESClient) AddTags(input *es.AddTagsInput) (*es.AddTagsOutput, error) {
	if aws.StringValue(input.TagList[0].Key) == "Ridecell-Operator" {
		m.mockDomainHasTags = true
	}
	return nil, nil
}
