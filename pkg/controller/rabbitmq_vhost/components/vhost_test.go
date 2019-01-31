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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"crypto/tls"
	"github.com/michaelklishin/rabbit-hole"
	"net/http"

	rmqvcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmq_vhost/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

type fakeRabbitClient struct {
	rmqvcomponents.RabbitMQManager
	FakeClient    *rabbithole.Client
	FakeVhostList []rabbithole.VhostInfo
}

func (frc *fakeRabbitClient) ListVhosts() ([]rabbithole.VhostInfo, error) {
	return frc.FakeVhostList, nil
}

func (frc *fakeRabbitClient) PutVhost(vhostname string, settings rabbithole.VhostSettings) (*http.Response, error) {
	var vhost_exists bool
	for _, element := range frc.FakeVhostList {
		if element.Name == vhostname {
			vhost_exists = true
		}
	}
	if !vhost_exists {
		frc.FakeVhostList = append(frc.FakeVhostList, rabbithole.VhostInfo{Name: vhostname, Tracing: false})
		return &http.Response{StatusCode: 201}, nil
	}
	return &http.Response{StatusCode: 200}, nil
}

var _ = Describe("RabbitmqVhost Vhost Component", func() {
	It("Reconcile with empty parameters", func() {
		comp := rmqvcomponents.NewVhost()
		instance.Spec.VhostName = ""
		instance.Spec.Connection.Password.Key = ""
		instance.Spec.Connection.Username = ""
		instance.Spec.Connection.Host = ""
		fakeFunc := func(uri string, user string, pass string, t *http.Transport) (rmqvcomponents.RabbitMQManager, error) {
			var mgr rmqvcomponents.RabbitMQManager
			mgr = &fakeRabbitClient{}
			return mgr, nil
		}
		comp.InjectFakeNewTLSClient(fakeFunc)
		Expect(comp).To(ReconcileContext(ctx))
	})
	It("Create new vhost if it does not exist", func() {
		comp := rmqvcomponents.NewVhost()
		mgr := &fakeRabbitClient{}
		fakeFunc := func(uri string, user string, pass string, t *http.Transport) (rmqvcomponents.RabbitMQManager, error) {
			fclient := &rabbithole.Client{Endpoint: uri, Username: user, Password: pass}
			mgr.FakeClient = fclient
			mgr.FakeVhostList = []rabbithole.VhostInfo{}
			return mgr, nil
		}
		comp.InjectFakeNewTLSClient(fakeFunc)
		transport := &http.Transport{TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // test server certificate is not trusted in case of self hosted rabbitmq
		},
		}
		comp.Client("test", "guest", "guest", transport)
		resp, _ := mgr.PutVhost("test", rabbithole.VhostSettings{Tracing: false})
		Expect(resp.StatusCode).To(Equal(201))
		elem := rabbithole.VhostInfo{Name: "test", Tracing: false}
		Expect(mgr.FakeVhostList).Should(ConsistOf(elem))
		Expect(mgr.FakeVhostList).To(HaveLen(1))
	})
	It("Fails to connect to unavailable rabbitmq host", func() {
		comp := rmqvcomponents.NewVhost()
		Expect(comp).ToNot(ReconcileContext(ctx))
	})
	It("Fails to create a rabbithole Client", func() {
		comp := rmqvcomponents.NewVhost()
		instance.Spec.Connection.Host = "htt://127.0.0.1:80"
		Expect(comp).ToNot(ReconcileContext(ctx))
	})
})
