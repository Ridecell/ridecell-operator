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

package v1beta1_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var _ = Describe("SummonPlatform types", func() {
	var helpers *test_helpers.PerTestHelpers

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("can create a SummonPlatform object", func() {
		c := helpers.Client
		key := types.NamespacedName{
			Name:      "foo",
			Namespace: helpers.Namespace,
		}
		created := &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: helpers.Namespace,
			}}
		fetched := &summonv1beta1.SummonPlatform{}
		err := c.Create(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		err = c.Get(context.TODO(), key, fetched)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetched.Spec).To(Equal(created.Spec))
	})

	It("can update a SummonPlatform object", func() {
		c := helpers.Client

		key := types.NamespacedName{
			Name:      "foo",
			Namespace: helpers.Namespace,
		}
		created := &summonv1beta1.SummonPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: helpers.Namespace,
			}}
		fetched := &summonv1beta1.SummonPlatform{}
		err := c.Create(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		created.Labels = map[string]string{"hello": "world"}
		err = c.Update(context.TODO(), created)
		Expect(err).NotTo(HaveOccurred())

		err = c.Get(context.TODO(), key, fetched)
		Expect(err).NotTo(HaveOccurred())
		Expect(fetched.Labels).To(Equal(created.Labels))
	})

	Describe("parsing unstructured config data", func() {
		It("can parse unstructured string data", func() {
			c := helpers.Client
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "summon.ridecell.io/v1beta1",
					"kind":       "SummonPlatform",
					"metadata": map[string]interface{}{
						"name":      "foo",
						"namespace": helpers.Namespace,
					},
					"spec": map[string]interface{}{
						"version": "1",
						"secrets": []string{"a"},
						"config": map[string]interface{}{
							"foo": "bar",
						},
					},
				},
			}

			err := c.Create(context.TODO(), obj)
			Expect(err).NotTo(HaveOccurred())

			fetched := &summonv1beta1.SummonPlatform{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: "foo", Namespace: helpers.Namespace}, fetched)
			Expect(err).NotTo(HaveOccurred())
			Expect(fetched.Spec.Config).To(HaveKey("foo"))
			Expect(fetched.Spec.Config["foo"].Bool).To(BeNil())
			Expect(fetched.Spec.Config["foo"].Float).To(BeNil())
			Expect(fetched.Spec.Config["foo"].String).To(PointTo(Equal("bar")))
		})

		It("can parse unstructured float data", func() {
			c := helpers.Client
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "summon.ridecell.io/v1beta1",
					"kind":       "SummonPlatform",
					"metadata": map[string]interface{}{
						"name":      "foo",
						"namespace": helpers.Namespace,
					},
					"spec": map[string]interface{}{
						"version": "1",
						"secrets": []string{"a"},
						"config": map[string]interface{}{
							"foo": 1234,
						},
					},
				},
			}

			err := c.Create(context.TODO(), obj)
			Expect(err).NotTo(HaveOccurred())

			fetched := &summonv1beta1.SummonPlatform{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: "foo", Namespace: helpers.Namespace}, fetched)
			Expect(err).NotTo(HaveOccurred())
			Expect(fetched.Spec.Config).To(HaveKey("foo"))
			Expect(fetched.Spec.Config["foo"].Bool).To(BeNil())
			Expect(fetched.Spec.Config["foo"].Float).To(PointTo(BeEquivalentTo(1234)))
			Expect(fetched.Spec.Config["foo"].String).To(BeNil())
		})

		It("can parse unstructured bool data", func() {
			c := helpers.Client
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "summon.ridecell.io/v1beta1",
					"kind":       "SummonPlatform",
					"metadata": map[string]interface{}{
						"name":      "foo",
						"namespace": helpers.Namespace,
					},
					"spec": map[string]interface{}{
						"version": "1",
						"secrets": []string{"a"},
						"config": map[string]interface{}{
							"foo": false,
						},
					},
				},
			}

			err := c.Create(context.TODO(), obj)
			Expect(err).NotTo(HaveOccurred())

			fetched := &summonv1beta1.SummonPlatform{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: "foo", Namespace: helpers.Namespace}, fetched)
			Expect(err).NotTo(HaveOccurred())
			Expect(fetched.Spec.Config).To(HaveKey("foo"))
			Expect(fetched.Spec.Config["foo"].Bool).To(PointTo(Equal(false)))
			Expect(fetched.Spec.Config["foo"].Float).To(BeNil())
			Expect(fetched.Spec.Config["foo"].String).To(BeNil())
		})

		It("can parse a bunch of unstructured data", func() {
			c := helpers.Client
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "summon.ridecell.io/v1beta1",
					"kind":       "SummonPlatform",
					"metadata": map[string]interface{}{
						"name":      "foo",
						"namespace": helpers.Namespace,
					},
					"spec": map[string]interface{}{
						"version": "1",
						"secrets": []string{"a"},
						"config": map[string]interface{}{
							"AMAZON_S3_USED":      true,
							"AWS_REGION":          "eu-central-1",
							"GOOGLE_ANALYTICS_ID": "UA-2345",
							"SESSION_COOKIE_AGE":  1,
						},
						"backup": map[string]interface{}{
							"ttl":            300000000000,
							"waitUntilReady": true,
						},
					},
				},
			}

			err := c.Create(context.TODO(), obj)
			Expect(err).NotTo(HaveOccurred())

			fetched := &summonv1beta1.SummonPlatform{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: "foo", Namespace: helpers.Namespace}, fetched)
			Expect(err).NotTo(HaveOccurred())
			Expect(fetched.Spec.Config).To(HaveKey("AMAZON_S3_USED"))
			Expect(fetched.Spec.Config["AMAZON_S3_USED"].Bool).To(PointTo(Equal(true)))
			Expect(fetched.Spec.Config["AMAZON_S3_USED"].Float).To(BeNil())
			Expect(fetched.Spec.Config["AMAZON_S3_USED"].String).To(BeNil())
			Expect(fetched.Spec.Config).To(HaveKey("AWS_REGION"))
			Expect(fetched.Spec.Config["AWS_REGION"].Bool).To(BeNil())
			Expect(fetched.Spec.Config["AWS_REGION"].Float).To(BeNil())
			Expect(fetched.Spec.Config["AWS_REGION"].String).To(PointTo(Equal("eu-central-1")))
			Expect(fetched.Spec.Config).To(HaveKey("GOOGLE_ANALYTICS_ID"))
			Expect(fetched.Spec.Config["GOOGLE_ANALYTICS_ID"].Bool).To(BeNil())
			Expect(fetched.Spec.Config["GOOGLE_ANALYTICS_ID"].Float).To(BeNil())
			Expect(fetched.Spec.Config["GOOGLE_ANALYTICS_ID"].String).To(PointTo(Equal("UA-2345")))
			Expect(fetched.Spec.Config).To(HaveKey("SESSION_COOKIE_AGE"))
			Expect(fetched.Spec.Config["SESSION_COOKIE_AGE"].Bool).To(BeNil())
			Expect(fetched.Spec.Config["SESSION_COOKIE_AGE"].Float).To(PointTo(BeEquivalentTo(1)))
			Expect(fetched.Spec.Config["SESSION_COOKIE_AGE"].String).To(BeNil())
			Expect(fetched.Spec.Backup.TTL).To(Equal(time.Minute * 5))
			Expect(*fetched.Spec.Backup.WaitUntilReady).To(BeTrue())
		})
	})
})
