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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	rabbithole "github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	rmqucomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmquser/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_rabbitmq"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("RabbitmqUser User Component", func() {
	comp := rmqucomponents.NewUser()
	var frc *fake_rabbitmq.FakeRabbitClient
	spec1 := dbv1beta1.RabbitmqUserSpec{
		Username: "rabbitmq-user-test1",
		Tags:     "policymaker",
		Permissions: []dbv1beta1.RabbitmqPermission{
			{
				Vhost:     "rabbimq-test1",
				Configure: ".*",
				Read:      ".*",
				Write:     ".*",
			},
			{
				Vhost:     "rabbitmq-test2",
				Configure: ".*",
				Read:      ".*",
				Write:     ".*",
			},
		},
	}

	BeforeEach(func() {
		// Set password in secrets
		rabbitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("rabbitmqpass"),
			},
		}
		ctx.Client = fake.NewFakeClient(instance, rabbitSecret)

		instance.Status.Connection.PasswordSecretRef = helpers.SecretRef{Name: "foo.rabbitmq-user-password", Key: "password"}

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
	It("creates permission for a user", func() {
		instance.Spec = spec1
		Expect(comp).To(ReconcileContext(ctx))
		Expect(frc.Permissions["rabbitmq-user-test1"]).To(HaveLen(2))
	})
	It("updates existing permissions for a user and vhost", func() {
		frc.Permissions["rabbitmq-user-test1"] = append(frc.Permissions["rabbitmq-user-test1"], rabbithole.PermissionInfo{
			Vhost:     "rabbitmq-test1",
			User:      "rabbitmq-user-test1",
			Configure: "abc.*",
			Read:      ".*",
			Write:     ".*",
		})
		instance.Spec = spec1
		Expect(comp).To(ReconcileContext(ctx))
		Expect(frc.Permissions["rabbitmq-user-test1"][1].Configure).To(Equal(".*"))
	})
	It("removes unwanted permissions for a user and vhost", func() {
		frc.Permissions["rabbitmq-user-test1"] = append(frc.Permissions["rabbitmq-user-test1"], rabbithole.PermissionInfo{
			Vhost:     "rabbitmq-test3",
			User:      "rabbitmq-user-test1",
			Configure: ".*",
			Read:      ".*",
			Write:     ".*",
		})
		instance.Spec = spec1
		Expect(comp).To(ReconcileContext(ctx))
		for key := range frc.Permissions["rabbitmq-user-test1"] {
			Expect(frc.Permissions["rabbitmq-user-test1"][key].Vhost).ToNot(Equal("rabbitmq-test3"))
		}
	})
})
