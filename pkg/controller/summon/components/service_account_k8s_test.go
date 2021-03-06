/*
Copyright 2020 Ridecell, Inc.

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

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("SummonPlatform iamrole Component", func() {

	BeforeEach(func() {
		os.Setenv("PERMISSIONS_BOUNDARY_ARN", "arn::123456789:test*")
		instance.Spec.UseIamRole = true
		instance.Spec.Environment = "dev"
	})

	It("creates an k8s serviceaccount object", func() {
		comp := summoncomponents.NewserviceAccountK8s()
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.ServiceAccount{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should not create serviceaccount object", func() {
		instance.Spec.UseIamRole = false
		comp := summoncomponents.NewserviceAccountK8s()
		Expect(comp).To(ReconcileContext(ctx))
		target := &corev1.ServiceAccount{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, target)
		Expect(err).To(HaveOccurred())
	})

})
