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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform s3bucket Component", func() {

	It("creates an S3Bucket object", func() {
		comp := summoncomponents.NewS3Bucket("aws/staticbucket.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &awsv1beta1.S3Bucket{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, target)
		Expect(err).ToNot(HaveOccurred())
		// Make sure it doesn't touch the MIV status.
		Expect(instance.Status.MIV.Bucket).To(Equal(""))
	})

	It("creates an MIV S3 bucket", func() {
		comp := summoncomponents.NewMIVS3Bucket("aws/mivbucket.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))
		target := &awsv1beta1.S3Bucket{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-miv", Namespace: "summon-dev"}, target)
		Expect(err).ToNot(HaveOccurred())
		Expect(instance.Status.MIV.Bucket).To(Equal("ridecell-foo-dev-miv"))
	})

	Context("when using an external MIV bucket", func() {
		BeforeEach(func() {
			instance.Spec.MIV.ExistingBucket = "asdf"
		})

		It("doesn't create a bucket", func() {
			comp := summoncomponents.NewMIVS3Bucket("aws/mivbucket.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			target := &awsv1beta1.S3Bucket{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-miv", Namespace: "summon-dev"}, target)
			Expect(err).To(HaveOccurred())
			Expect(kerrors.IsNotFound(err)).To(BeTrue())
			Expect(instance.Status.MIV.Bucket).To(Equal("asdf"))
		})

		It("deletes an existing operator-controlled bucket", func() {
			target := &awsv1beta1.S3Bucket{ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-miv", Namespace: "summon-dev"}}
			ctx.Client = fake.NewFakeClient(instance, target)
			comp := summoncomponents.NewMIVS3Bucket("aws/mivbucket.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-miv", Namespace: "summon-dev"}, target)
			Expect(err).To(HaveOccurred())
			Expect(kerrors.IsNotFound(err)).To(BeTrue())
			Expect(instance.Status.MIV.Bucket).To(Equal("asdf"))
		})

		It("test our temporary region respect hack static", func() {
			newS3Bucket := &awsv1beta1.S3Bucket{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dev",
					Namespace: "summon-dev",
				},
				Spec: awsv1beta1.S3BucketSpec{
					Region: "notouchy",
				},
			}
			ctx.Client = fake.NewFakeClient(newS3Bucket)
			comp := summoncomponents.NewS3Bucket("aws/staticbucket.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			target := &awsv1beta1.S3Bucket{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target.Spec.Region).To(Equal("notouchy"))
		})

		It("test our temporary region respect hack miv", func() {
			newS3Bucket := &awsv1beta1.S3Bucket{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo-dev",
					Namespace: "summon-dev",
				},
				Spec: awsv1beta1.S3BucketSpec{
					Region: "notouchy",
				},
			}
			ctx.Client = fake.NewFakeClient(newS3Bucket)
			comp := summoncomponents.NewS3Bucket("aws/mivbucket.yml.tpl")
			Expect(comp).To(ReconcileContext(ctx))
			target := &awsv1beta1.S3Bucket{}
			err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev", Namespace: "summon-dev"}, target)
			Expect(err).ToNot(HaveOccurred())
			Expect(target.Spec.Region).To(Equal("notouchy"))
		})
	})
})
