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
	"context"
	"os"

	. "github.com/Benjamintf1/unmarshalledmatchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform iamuser Component", func() {

	BeforeEach(func() {
		os.Setenv("PERMISSIONS_BOUNDARY_ARN", "arn::123456789:test*")
		instance.Spec.SQSQueue = "test-sqs-queue"
	})

	It("creates an IAMUser object", func() {
		comp := summoncomponents.NewIAMUser("aws/iamuser.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &awsv1beta1.IAMUser{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("cross region sqs", func() {
		instance.Spec.SQSRegion = "cross-region"
		comp := summoncomponents.NewIAMUser("aws/iamuser.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &awsv1beta1.IAMUser{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
		Expect(target.Spec.InlinePolicies["allow_sqs"]).To(ContainSubstring("arn:aws:sqs:cross-region:123456789:test-sqs-queue"))
	})

	Context("MIV policy", func() {
		It("handles an internal bucket", func() {
			comp := summoncomponents.NewIAMUser("aws/iamuser.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			target := &awsv1beta1.IAMUser{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target.Spec.InlinePolicies["allow_s3_miv"]).To(ContainOrderedJSON(`{"Statement": [{"Resource": "arn:aws:s3:::ridecell-foo-dev-miv"}]}`))
		})

		It("handles an external bucket", func() {
			instance.Spec.MIV.ExistingBucket = "asdf"
			comp := summoncomponents.NewIAMUser("aws/iamuser.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			target := &awsv1beta1.IAMUser{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target.Spec.InlinePolicies["allow_s3_miv"]).To(ContainOrderedJSON(`{"Statement": [{"Resource": "arn:aws:s3:::asdf"}]}`))
		})
	})
})
