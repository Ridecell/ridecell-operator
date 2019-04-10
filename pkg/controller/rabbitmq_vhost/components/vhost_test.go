/*
Copyright 2018 Ridecell, Inc.

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
	rabbithole "github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	rmqvcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmq_vhost/components"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers/fake_rabbitmq"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("RabbitmqVhost Vhost Component", func() {
	comp := rmqvcomponents.NewVhost()
	var frc *fake_rabbitmq.FakeRabbitClient

	BeforeEach(func() {
		user1 := &dbv1beta1.RabbitmqUser{
			ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
			Status: dbv1beta1.RabbitmqUserStatus{
				Status: dbv1beta1.StatusReady,
				Connection: dbv1beta1.RabbitmqStatusConnection{
					Username: "foo-user",
					PasswordSecretRef: helpers.SecretRef{
						Name: "foo.rabbitmq-user-password",
						Key:  "password",
					},
				},
			},
		}
		user2 := &dbv1beta1.RabbitmqUser{
			ObjectMeta: metav1.ObjectMeta{Name: "bar", Namespace: "default"},
			Status: dbv1beta1.RabbitmqUserStatus{
				Status: "",
			},
		}
		ctx.Client = fake.NewFakeClient(instance, user1, user2)

		comp = rmqvcomponents.NewVhost()
		frc = fake_rabbitmq.New()
		comp.InjectClientFactory(frc.Factory)
	})

	It("creates a new vhost if it does not exist", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(frc.Vhosts).To(HaveLen(1))
		Expect(frc.Vhosts[0].Name).To(Equal("foo"))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
		Expect(instance.Status.Connection.Host).To(Equal("mockhost"))
		Expect(instance.Status.Connection.Username).To(Equal("foo-user"))
		Expect(instance.Status.Connection.Vhost).To(Equal("foo"))
	})

	It("does not create a new vhost if it exists already", func() {
		frc.Vhosts = append(frc.Vhosts, rabbithole.VhostInfo{Name: "foo"})
		Expect(comp).To(ReconcileContext(ctx))
		Expect(frc.Vhosts).To(HaveLen(1))
		Expect(frc.Vhosts[0].Name).To(Equal("foo"))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
		Expect(instance.Status.Connection.Host).To(Equal("mockhost"))
		Expect(instance.Status.Connection.Username).To(Equal("foo-user"))
		Expect(instance.Status.Connection.Vhost).To(Equal("foo"))
	})

	It("does not report ready if the user is not ready", func() {
		instance.Name = "bar"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(""))
	})

	It("does report ready if the user is not ready and SkipUser is enabled", func() {
		instance.Name = "bar"
		instance.Spec.SkipUser = true
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(dbv1beta1.StatusReady))
	})
})
