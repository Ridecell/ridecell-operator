/*
Copyright 2019-2020 Ridecell, Inc.

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
	ingressv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/ingress/v1beta1"
	components "github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/controller/ridecellingress"
	ricomponents "github.com/Ridecell/ridecell-operator/pkg/controller/ridecellingress/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("RidecellIngress Component", func() {

	var instance ingressv1beta1.RidecellIngress

	BeforeEach(func() {
		instance = ingressv1beta1.RidecellIngress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ridecellingress-sample",
				Namespace: "default",
				Labels: map[string]string{
					"ridecell.io/environment": "sandbox",
					"ridecell.io/region":      "us",
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
		ctx = components.NewTestContext(&instance, ridecellingress.Templates)
	})

	It("creates an RidecellIngress object using above template", func() {
		comp := ricomponents.NewIngress()
		Expect(comp).To(ReconcileContext(ctx))
		target := &extv1beta1.Ingress{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
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

})
