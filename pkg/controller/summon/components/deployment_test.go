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
	"fmt"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("deployment Component", func() {

	BeforeEach(func() {
		instance.Status.Status = summonv1beta1.StatusDeploying
	})

	It("runs a basic reconcile", func() {
		comp := summoncomponents.NewDeployment("static/deployment.yml.tpl")

		// Set this value so created template does not contain a nil value
		numReplicas := int32(1)
		instance.Spec.StaticReplicas = &numReplicas

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace},
			Data:       map[string]string{"summon-platform.yml": "{}\n"},
		}

		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
			Data: map[string][]byte{
				"filler": []byte("test"),
				"test":   []byte("another_test"),
			},
		}

		ctx.Client = fake.NewFakeClient(appSecrets, configMap)
		Expect(comp).To(ReconcileContext(ctx))

		deployment := &appsv1.Deployment{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-static", Namespace: instance.Namespace}, deployment)
		Expect(err).ToNot(HaveOccurred())
		deploymentPodAnnotations := deployment.Spec.Template.Annotations
		Expect(deploymentPodAnnotations["summon.ridecell.io/appSecretsHash"]).To(HaveLen(40))
		Expect(deploymentPodAnnotations["summon.ridecell.io/configHash"]).To(HaveLen(40))
	})

	It("makes sure keys are sorted before hash", func() {
		comp := summoncomponents.NewDeployment("static/deployment.yml.tpl")

		// Set this value so created template does not contain a nil value
		numReplicas := int32(1)
		instance.Spec.StaticReplicas = &numReplicas

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace},
			Data:       map[string]string{"summon-platform.yml": "{}\n"},
		}

		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
			Data: map[string][]byte{
				"test":   []byte("another_test"),
				"filler": []byte("test"),
			},
		}

		ctx.Client = fake.NewFakeClient(appSecrets, configMap)
		Expect(comp).To(ReconcileContext(ctx))

		deployment := &appsv1.Deployment{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-static", Namespace: instance.Namespace}, deployment)
		Expect(err).ToNot(HaveOccurred())
		deploymentPodAnnotations := deployment.Spec.Template.Annotations
		Expect(deploymentPodAnnotations["summon.ridecell.io/appSecretsHash"]).To(HaveLen(40))
	})

	It("updates existing hashes", func() {
		comp := summoncomponents.NewDeployment("static/deployment.yml.tpl")

		// Set this value so created template does not contain a nil value
		numReplicas := int32(1)
		instance.Spec.StaticReplicas = &numReplicas

		// Create our first hashes
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace},
			Data:       map[string]string{"summon-platform.yml": "{}\n"},
		}

		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
			Data:       map[string][]byte{"filler": []byte("test")},
		}

		ctx.Client = fake.NewFakeClient(appSecrets, configMap)
		Expect(comp).To(ReconcileContext(ctx))

		// Create our second hashes
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace},
			Data:       map[string]string{"summon-platform.yml": "{test}\n"},
		}

		appSecrets = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
			Data:       map[string][]byte{"filler": []byte("test2")},
		}

		ctx.Client = fake.NewFakeClient(appSecrets, configMap)
		Expect(comp).To(ReconcileContext(ctx))

		deployment := &appsv1.Deployment{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-static", Namespace: instance.Namespace}, deployment)
		Expect(err).ToNot(HaveOccurred())
		deploymentPodAnnotations := deployment.Spec.Template.Annotations
		Expect(deploymentPodAnnotations["summon.ridecell.io/appSecretsHash"]).To(HaveLen(40))
		Expect(deploymentPodAnnotations["summon.ridecell.io/configHash"]).To(HaveLen(40))

	})

	It("creates an statefulset object using celerybeat template", func() {
		comp := summoncomponents.NewDeployment("celerybeat/statefulset.yml.tpl")

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace},
			Data:       map[string]string{"summon-platform.yml": "{}\n"},
		}

		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
			Data: map[string][]byte{
				"filler": []byte("test"),
				"test":   []byte("another_test"),
			},
		}

		ctx.Client = fake.NewFakeClient(appSecrets, configMap)
		Expect(comp).To(ReconcileContext(ctx))
		target := &appsv1.StatefulSet{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-celerybeat", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("sets celerybeat to 0 if NoCelerybeat is true", func() {
		comp := summoncomponents.NewDeployment("celerybeat/statefulset.yml.tpl")

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", instance.Name), Namespace: instance.Namespace},
			Data:       map[string]string{"summon-platform.yml": "{}\n"},
		}

		appSecrets := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.app-secrets", instance.Name), Namespace: instance.Namespace},
			Data: map[string][]byte{
				"filler": []byte("test"),
				"test":   []byte("another_test"),
			},
		}

		instance.Spec.NoCelerybeat = true

		ctx.Client = fake.NewFakeClient(appSecrets, configMap)
		Expect(comp).To(ReconcileContext(ctx))
		target := &appsv1.StatefulSet{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-celerybeat", Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
		Expect(target.Spec.Replicas).To(PointTo(BeEquivalentTo(0)))
	})
})
