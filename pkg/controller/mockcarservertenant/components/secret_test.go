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
	"github.com/Ridecell/ridecell-operator/pkg/components"
	mockcarservertenantcomponent "github.com/Ridecell/ridecell-operator/pkg/controller/mockcarservertenant/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("MockCarServerTenant Secret Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = mockcarservertenantcomponent.NewSecret()
	})

	It("creates a secret with a random data", func() {
		Expect(comp).To(ReconcileContext(ctx))
		secret := &corev1.Secret{}
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "test-dev.tenant-otakeys", Namespace: "summon-dev"}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(secret.Data["OTAKEYS_API_KEY"])).To(Equal("test-dev-api-key"))
		Expect(secret.Data["OTAKEYS_SECRET_KEY"]).To(HaveLen(32))
		Expect(secret.Data["OTAKEYS_TOKEN"]).To(HaveLen(32))
		Expect(secret.Data["OTAKEYS_PUSH_API_KEY"]).To(HaveLen(32))
		Expect(secret.Data["OTAKEYS_PUSH_SECRET_KEY"]).To(HaveLen(32))
		Expect(secret.Data["OTAKEYS_PUSH_TOKEN"]).To(HaveLen(32))
	})

	It("does not update an existing data", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "test-dev.tenant-otakeys", Namespace: "summon-dev"},
			Data: map[string][]byte{
				"OTAKEYS_API_KEY":         []byte("test-dev-api-key"),
				"OTAKEYS_SECRET_KEY":      []byte("1234567890poiuytrewqasdfghjklmnb"),
				"OTAKEYS_TOKEN":           []byte("1234567890poiuytrewqasdfghjklmnb"),
				"OTAKEYS_PUSH_API_KEY":    []byte("1234567890poiuytrewqasdfghjklmnb"),
				"OTAKEYS_PUSH_SECRET_KEY": []byte("1234567890poiuytrewqasdfghjklmnb"),
				"OTAKEYS_PUSH_TOKEN":      []byte("1234567890poiuytrewqasdfghjklmnb"),
			},
		}
		ctx.Client = fake.NewFakeClient(instance, secret)
		Expect(comp).To(ReconcileContext(ctx))
		err := ctx.Get(ctx.Context, types.NamespacedName{Name: "test-dev.tenant-otakeys", Namespace: "summon-dev"}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Data).To(HaveKeyWithValue("OTAKEYS_API_KEY", []byte("test-dev-api-key")))
		Expect(secret.Data).To(HaveKeyWithValue("OTAKEYS_SECRET_KEY", []byte("1234567890poiuytrewqasdfghjklmnb")))
		Expect(secret.Data).To(HaveKeyWithValue("OTAKEYS_TOKEN", []byte("1234567890poiuytrewqasdfghjklmnb")))
		Expect(secret.Data).To(HaveKeyWithValue("OTAKEYS_PUSH_API_KEY", []byte("1234567890poiuytrewqasdfghjklmnb")))
		Expect(secret.Data).To(HaveKeyWithValue("OTAKEYS_PUSH_SECRET_KEY", []byte("1234567890poiuytrewqasdfghjklmnb")))
		Expect(secret.Data).To(HaveKeyWithValue("OTAKEYS_PUSH_TOKEN", []byte("1234567890poiuytrewqasdfghjklmnb")))
	})

	It("fills in the secret info in the status", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.KeysSecretRef).To(Equal("test-dev.tenant-otakeys"))
	})
})
