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
	ingressv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/ingress/v1beta1"
	ricomponents "github.com/Ridecell/ridecell-operator/pkg/controller/ridecellingress/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("RidecellIngress Defaults Component", func() {

	It("fills out type variables with default values", func() {
		comp := ricomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(ingressv1beta1.RootDomain).To(Equal("ridecell.io"))
		Expect(ingressv1beta1.TLS_ACME).To(Equal("true"))
		Expect(ingressv1beta1.ClusterIssuer).To(Equal("letsencrypt-prod"))
		Expect(ingressv1beta1.IngressClass).To(Equal("traefik"))
	})
})
