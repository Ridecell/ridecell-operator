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
	"os"

	rabbithole "github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rmqucomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmquser/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_rabbitmq"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("RabbitmqUser Component", func() {
	comp := rmqucomponents.NewUser()
	var frc *fake_rabbitmq.FakeRabbitClient

	BeforeEach(func() {
		// Set password in secrets
		rabbitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: instance.Namespace},
			Data: map[string][]byte{
				"password": []byte("rabbitmqpass"),
			},
		}
		ctx.Client = fake.NewFakeClient(instance, rabbitSecret)

		comp = rmqucomponents.NewUser()
		frc = fake_rabbitmq.New()
		comp.InjectClientFactory(frc.Factory)
	})

	It("Create new user if it does not exist", func() {
		instance.Spec.Username = "foo"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(frc.Users).To(HaveLen(1))
		Expect(frc.Users[0].Name).To(Equal("foo"))
		Expect(frc.Users[0].PasswordHash).To(Equal("rabbitmqpass"))
	})

	It("Does not create a user if they already exist", func() {
		frc.Users = append(frc.Users, rabbithole.UserInfo{Name: "foo", PasswordHash: "asdf"})
		instance.Spec.Username = "foo"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(frc.Users).To(HaveLen(1))
		Expect(frc.Users[0].Name).To(Equal("foo"))
	})

	It("Handles an invalid URI", func() {
		os.Setenv("RABBITMQ_URI", "htt://127.0.0.1:80")
		Expect(comp).ToNot(ReconcileContext(ctx))
	})

	It("set the output connection info", func() {
		instance.Spec.Username = "foo"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Connection.Host).To(Equal("mockhost"))
		Expect(instance.Status.Connection.Port).To(Equal(5671))
	})
})
