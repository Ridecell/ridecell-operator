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

package fake_rabbitmq

import (
	"net/http"

	rabbithole "github.com/michaelklishin/rabbit-hole"

	"github.com/Ridecell/ridecell-operator/pkg/utils"
)

type FakeRabbitClient struct {
	Users  []rabbithole.UserInfo
	Vhosts []rabbithole.VhostInfo
}

func New() *FakeRabbitClient {
	return &FakeRabbitClient{Users: []rabbithole.UserInfo{}, Vhosts: []rabbithole.VhostInfo{}}
}

func (frc *FakeRabbitClient) Inject(_uri, _user, _pass string, _t *http.Transport) (utils.RabbitMQManager, error) {
	return frc, nil
}

func (frc *FakeRabbitClient) ListUsers() ([]rabbithole.UserInfo, error) {
	return frc.Users, nil
}

func (frc *FakeRabbitClient) PutUser(username string, settings rabbithole.UserSettings) (*http.Response, error) {
	for _, user := range frc.Users {
		if user.Name == username {
			user.PasswordHash = settings.Password
			return &http.Response{StatusCode: 200}, nil
		}
	}
	frc.Users = append(frc.Users, rabbithole.UserInfo{Name: username, PasswordHash: settings.Password})
	return &http.Response{StatusCode: 201}, nil
}

func (frc *FakeRabbitClient) ListVhosts() ([]rabbithole.VhostInfo, error) {
	return frc.Vhosts, nil
}

func (frc *FakeRabbitClient) PutVhost(_vhost string, _settings rabbithole.VhostSettings) (*http.Response, error) {
	// TODO
	return &http.Response{StatusCode: 500}, nil
}
