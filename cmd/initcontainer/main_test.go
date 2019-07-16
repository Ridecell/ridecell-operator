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

package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	main "github.com/Ridecell/ridecell-operator/cmd/initcontainer"
	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
)

var _ = Describe("InitContainer", func() {
	It("should add the broker URL", func() {
		rmqv := &dbv1beta1.RabbitmqVhost{
			ObjectMeta: metav1.ObjectMeta{Name: "svc-us-prod-dispatch", Namespace: "dispatch"},
			Status: dbv1beta1.RabbitmqVhostStatus{
				Connection: dbv1beta1.RabbitmqStatusConnection{
					Host:     "mybunny",
					Username: "svc-us-prod-dispatch-user",
					Vhost:    "svc-us-prod-dispatch-user",
					PasswordSecretRef: helpers.SecretRef{
						Name: "my-user-secret",
						Key:  "password",
					},
				},
			},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "my-user-secret", Namespace: "dispatch"},
			Data: map[string][]byte{
				"password": []byte("topsecret"),
			},
		}
		c := fake.NewFakeClient(rmqv, secret)

		data := map[string]interface{}{}
		err := main.Update("us-prod", "dispatch", c, data)
		Expect(err).ToNot(HaveOccurred())
		Expect(data).To(HaveKeyWithValue("CELERY_BROKER_URL", "pyamqp://svc-us-prod-dispatch-user:topsecret@mybunny/svc-us-prod-dispatch-user?ssl=true"))
	})
})
