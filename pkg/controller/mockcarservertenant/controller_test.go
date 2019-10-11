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

package mockcarservertenant_test

import (
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_mockcarserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"time"
)

const timeout = time.Second * 30

var _ = Describe("mockcarservertenant controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var instance *summonv1beta1.MockCarServerTenant
	fake_mockcarserver.Run()

	BeforeEach(func() {
		os.Setenv("MOCKCARSERVER_URI", "http://localhost:9090")
		os.Setenv("MOCKCARSERVER_AUTH", "1234567890")
		helpers = testHelpers.SetupTest()
		instance = &summonv1beta1.MockCarServerTenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo-sample",
				Namespace: helpers.Namespace,
			},
			Spec: summonv1beta1.MockCarServerTenantSpec{
				TenantHardwareType: "OTAKEYS",
			},
		}
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			helpers.DebugList(&summonv1beta1.MockCarServerTenant{})
			helpers.DebugList(&corev1.Secret{})
		}
		helpers.TeardownTest()
	})

	It("Creates MockCarServerTenant kind", func() {
		c := helpers.TestClient
		c.Create(instance)
		target := &summonv1beta1.MockCarServerTenant{}
		c.EventuallyGet(helpers.Name("foo-sample"), target, c.EventuallyStatus("Success"))
		// Check for otakeys secret
		secret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("foo-sample.tenant-otakeys"), secret)
	})

	It("Deletes MockCarServerTenant kind", func() {
		c := helpers.TestClient
		c.Create(instance)
		target := &summonv1beta1.MockCarServerTenant{}
		c.EventuallyGet(helpers.Name("foo-sample"), target, c.EventuallyStatus("Success"))
		// Delete the instance and check for secret
		c.Delete(instance)
		secret := &corev1.Secret{}
		Eventually(func() error {
			return helpers.Client.Get(context.TODO(), helpers.Name("foo-sample.tenant-otakeys"), secret)
		}).ShouldNot(Succeed())
	})
})
