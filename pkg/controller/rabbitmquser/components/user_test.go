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
	"github.com/michaelklishin/rabbit-hole"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"

	"crypto/sha512"
	"encoding/hex"
	rmqucomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmquser/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	"github.com/Ridecell/ridecell-operator/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeRabbitClient struct {
	utils.RabbitMQManager
	FakeClient   *rabbithole.Client
	FakeUserList []rabbithole.UserInfo
}

func (frc *fakeRabbitClient) ListUsers() ([]rabbithole.UserInfo, error) {
	return frc.FakeUserList, nil
}

func (frc *fakeRabbitClient) PutUser(username string, settings rabbithole.UserSettings) (*http.Response, error) {
	sha_512 := sha512.New()
	_, err := sha_512.Write([]byte(settings.Password))
	if err != nil {
		return &http.Response{StatusCode: 500}, errors.Wrapf(err, "error writing bytes")
	}
	passInBytes := sha_512.Sum(nil)
	passHash := hex.EncodeToString(passInBytes)
	var user_exists bool
	var usr rabbithole.UserInfo
	for _, element := range frc.FakeUserList {
		if element.Name == username {
			user_exists = true
			usr = element
		}
	}
	if !user_exists {
		frc.FakeUserList = append(frc.FakeUserList, rabbithole.UserInfo{Name: username, PasswordHash: passHash})
		return &http.Response{StatusCode: 201}, nil
	}
	usr.PasswordHash = passHash
	return &http.Response{StatusCode: 200}, nil
}

var _ = Describe("RabbitmqUser Component", func() {
	BeforeEach(func() {
		// Set password in secrets
		dbSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "admin.foo-rabbitmq.credentials", Namespace: "default"},
			Data: map[string][]byte{
				"password": []byte("secretrabbitmqpass"),
			},
		}
		rabbitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: instance.Namespace},
			Data: map[string][]byte{
				"password": []byte("rabbitmqpass"),
			},
		}
		ctx.Client = fake.NewFakeClient(dbSecret, rabbitSecret)
	})

	It("Reconcile with empty username", func() {
		comp := rmqucomponents.NewUser()
		instance.Spec.Username = ""
		os.Setenv("RABBITMQ_HOST_DEV", "https://rabbitmq-prod:5671")
		os.Setenv("RABBITMQ_SUPERUSER", "rabbitmq-superuser")
		os.Setenv("RABBITMQ_SUPERUSER_PASSWORD", "rabbitmq-superuser-password")
		fakeFunc := func(uri string, user string, pass string, t *http.Transport) (utils.RabbitMQManager, error) {
			var mgr utils.RabbitMQManager = &fakeRabbitClient{}
			return mgr, nil
		}
		comp.InjectFakeNewTLSClient(fakeFunc)
		Expect(comp).To(ReconcileContext(ctx))
	})
	It("Create new user if it does not exist", func() {
		comp := rmqucomponents.NewUser()
		instance.Spec.Username = "foo"
		os.Setenv("RABBITMQ_HOST_DEV", "https://rabbitmq-prod:5671")
		os.Setenv("RABBITMQ_SUPERUSER", "rabbitmq-superuser")
		os.Setenv("RABBITMQ_SUPERUSER_PASSWORD", "rabbitmq-superuser-password")
		mgr := &fakeRabbitClient{}
		fakeFunc := func(uri string, user string, pass string, t *http.Transport) (utils.RabbitMQManager, error) {
			fclient := &rabbithole.Client{Endpoint: uri, Username: user, Password: pass}
			mgr.FakeClient = fclient
			mgr.FakeUserList = []rabbithole.UserInfo{}
			return mgr, nil
		}
		comp.InjectFakeNewTLSClient(fakeFunc)
		Expect(comp).To(ReconcileContext(ctx))
		Expect(mgr.FakeUserList).To(HaveLen(1))
		Ω(mgr.FakeUserList[0].Name).Should(Equal("foo"))
	})
	It("Fails to connect to unavailable rabbitmq host", func() {
		comp := rmqucomponents.NewUser()
		Expect(comp).ToNot(ReconcileContext(ctx))
	})
	It("Fails to create a rabbithole Client", func() {
		os.Setenv("RABBITMQ_HOST_DEV", "htt://127.0.0.1:80")
		comp := rmqucomponents.NewUser()
		Expect(comp).ToNot(ReconcileContext(ctx))
	})
})
