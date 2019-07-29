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

package ridecellingress_test

import (
	ingressv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/ingress/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ridecellingress controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *ingressv1beta1.RidecellIngress
	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		// Define RidecellIngress instance for the test
		instance = &ingressv1beta1.RidecellIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ridecellingress-sample",
				Namespace: helpers.Namespace,
				Labels: map[string]string{
					"ridecell.io/environment": "sandbox",
					"ridecell.io/region":      "us",
					"ping":                    "pong",
				},
				Annotations: map[string]string{
					"kubernetes.io/ingress.class": "nginx",
					"kubernetes.io/tls-acme":      "false",
					"abc.io/ping":                 "pong",
				},
			},
			Spec: extv1beta1.IngressSpec{
				Rules: []extv1beta1.IngressRule{
					{
						Host: "hostname1",
					},
					{
						Host: "hostname2.custom.domain.com",
					},
				},
				TLS: []extv1beta1.IngressTLS{
					{
						Hosts: []string{
							"hostname1",
							"hostname2.custom.domain.com",
						},
						SecretName: "custom-tls",
					},
				},
			},
		}
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&ingressv1beta1.RidecellIngressList{})
			helpers.DebugList(&extv1beta1.Ingress{})
		}
		helpers.TeardownTest()
	})

	It("Creates RidecellIngress kind", func() {
		c := helpers.TestClient
		c.Create(instance)
		// Test the Ingress object created by RidecellIngress
		target := &extv1beta1.Ingress{}
		c.EventuallyGet(helpers.Name("ridecellingress-sample"), target)
		//Check for full domain names
		Expect(target.Spec.Rules[0].Host).To(HaveSuffix("ridecell.io"))
		Expect(target.Spec.Rules[1].Host).ToNot(HaveSuffix("ridecell.io"))
		Expect(target.Spec.TLS[0].Hosts[0]).To(HaveSuffix("ridecell.io"))
		Expect(target.Spec.TLS[0].Hosts[1]).ToNot(HaveSuffix("ridecell.io"))
		// Check for annotations and its values on target
		Expect(target.Annotations).To(HaveKeyWithValue("kubernetes.io/ingress.class", "nginx"))
		Expect(target.Annotations).To(HaveKeyWithValue("kubernetes.io/tls-acme", "false"))
		// Check for custom annotation
		Expect(target.Annotations).To(HaveKeyWithValue("abc.io/ping", "pong"))
		// below annotation should be added automatically with default value as its not present in instance definition
		Expect(target.Annotations).To(HaveKeyWithValue("certmanager.k8s.io/cluster-issuer", "letsencrypt-prod"))
	})

	It("Creates RidecellIngress kind with invalid hostname", func() {
		c := helpers.TestClient
		localinstance := instance
		localinstance.Name = "ri-invalid-host"
		//Modify instance hostname to invalid value
		localinstance.Spec.Rules[0].Host = "hostname!@#"
		c.Create(localinstance)
		target := &ingressv1beta1.RidecellIngress{}
		//The status should be Error
		c.EventuallyGet(helpers.Name("ri-invalid-host"), target, c.EventuallyStatus("Error"))
		//Modifying object to valid value, the status should be Success
		target.Spec.Rules[0].Host = "hostname"
		c.Update(target)
		c.EventuallyGet(helpers.Name("ri-invalid-host"), target, c.EventuallyStatus("Success"))

	})

	It("Adds new label and annotation to instance and verifies", func() {
		c := helpers.TestClient
		localinstance := instance
		localinstance.Name = "ri-add"
		c.Create(localinstance)
		target := &ingressv1beta1.RidecellIngress{}
		ingresstarget := &extv1beta1.Ingress{}
		c.EventuallyGet(helpers.Name("ri-add"), target, c.EventuallyStatus("Success"))

		//Adding new label and annotation to RidecellIngress Object
		target.Annotations["newAnnotation"] = "Yuhoo"
		target.Labels["newLabel"] = "Huyaa"

		c.Update(target)
		//Wait for changes to commplete
		c.EventuallyGet(helpers.Name("ri-add"), target, c.EventuallyStatus("Success"))
		//Check on Ingress object for new annotation and label
		c.EventuallyGet(helpers.Name("ri-add"), ingresstarget)
		Expect(ingresstarget.Annotations).To(HaveKeyWithValue("newAnnotation", "Yuhoo"))
		Expect(ingresstarget.Labels).To(HaveKeyWithValue("newLabel", "Huyaa"))
	})

	It("Deletes a label and annotation to instance and verifies", func() {
		c := helpers.TestClient
		localinstance := instance
		localinstance.Name = "ri-delete"
		c.Create(localinstance)
		target := &ingressv1beta1.RidecellIngress{}
		ingresstarget := &extv1beta1.Ingress{}
		c.EventuallyGet(helpers.Name("ri-delete"), target, c.EventuallyStatus("Success"))

		//Remove label and annotation to RidecellIngress Object
		delete(target.Annotations, "abc.io/ping")
		delete(target.Labels, "ping")

		c.Update(target)
		//Wait for changes to commplete
		c.EventuallyGet(helpers.Name("ri-delete"), target, c.EventuallyStatus("Success"))
		//Check on Ingress object for deleted annotation and label
		c.EventuallyGet(helpers.Name("ri-delete"), ingresstarget)
		Expect(ingresstarget.Annotations).ToNot(HaveKey("abc.io/ping"))
		Expect(ingresstarget.Labels).ToNot(HaveKey("ping"))
	})
})
