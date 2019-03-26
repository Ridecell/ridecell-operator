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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("rotate_fernet Component", func() {

	BeforeEach(func() {
		timeDuration, _ := time.ParseDuration("8760h")
		instance.Spec.FernetKeyLifetime = timeDuration
	})

	It("create a new secret if not present", func() {
		comp := summoncomponents.NewFernetRotate()
		ctx.Client = fake.NewFakeClient()

		Expect(comp.IsReconcilable(ctx)).To(Equal(true))
		Expect(comp).To(ReconcileContext(ctx))

		fernetSecret := &corev1.Secret{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s.fernet-keys", instance.Name), Namespace: instance.Namespace}, fernetSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(fernetSecret.Data).To(HaveLen(1))
		for _, v := range fernetSecret.Data {
			Expect(v).To(HaveLen(86))
		}
	})

	It("Adds a new key if the old one is expired", func() {
		comp := summoncomponents.NewFernetRotate()

		timeDuration, _ := time.ParseDuration("-8761h")
		timeNow := time.Now()
		negativeTime := timeNow.Add(timeDuration)
		timeStamp := time.Time.Format(negativeTime, summoncomponents.CustomTimeLayout)

		fernetSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.fernet-keys", instance.Name), Namespace: instance.Namespace},
			Data: map[string][]byte{
				timeStamp: []byte("SfrtdqmOy+KTaKfLy8Cr62R9HWPEHRu+xr7Vo/Ld1EMHy4omdUUgvJ/ke+PikYthTh7lnrYeQpD3e8EUK1WhEg")},
		}
		ctx.Client = fake.NewFakeClient(fernetSecret)
		Expect(comp.IsReconcilable(ctx)).To(Equal(true))
		Expect(comp).To(ReconcileContext(ctx))

		fetchSecret := &corev1.Secret{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("%s.fernet-keys", instance.Name), Namespace: instance.Namespace}, fetchSecret)
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchSecret.Data).To(HaveLen(2))
	})
})
