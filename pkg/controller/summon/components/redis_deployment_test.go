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

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	appsv1 "k8s.io/api/apps/v1"
)

var _ = Describe("redis_deployment Component", func() {

	It("creates a redis deployment", func() {
		instance.Status.Status = summonv1beta1.StatusDeploying
		comp := summoncomponents.NewRedisDeployment("redis/deployment.yml.tpl")
		Expect(comp).To(ReconcileContext(ctx))

		deployment := &appsv1.Deployment{}
		err := ctx.Get(context.TODO(), types.NamespacedName{Name: "foo-redis", Namespace: instance.Namespace}, deployment)
		Expect(err).ToNot(HaveOccurred())
	})
})
