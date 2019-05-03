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

	"github.com/Ridecell/ridecell-operator/pkg/utils"
	rabbithole "github.com/michaelklishin/rabbit-hole"
)

type FakeRabbitClient struct {
	Users       []rabbithole.UserInfo
	Vhosts      []rabbithole.VhostInfo
	Policies    map[string][]rabbithole.Policy
	Permissions map[string][]rabbithole.PermissionInfo
}

func New() *FakeRabbitClient {
	return &FakeRabbitClient{
		Users:       []rabbithole.UserInfo{},
		Vhosts:      []rabbithole.VhostInfo{},
		Policies:    make(map[string][]rabbithole.Policy),
		Permissions: make(map[string][]rabbithole.PermissionInfo),
	}
}

func (frc *FakeRabbitClient) Factory(_uri, _user, _pass string, _t *http.Transport) (utils.RabbitMQManager, error) {
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

func (frc *FakeRabbitClient) PutVhost(vhost string, _settings rabbithole.VhostSettings) (*http.Response, error) {
	for _, element := range frc.Vhosts {
		if element.Name == vhost {
			return &http.Response{StatusCode: 200}, nil
		}
	}
	frc.Vhosts = append(frc.Vhosts, rabbithole.VhostInfo{Name: vhost})
	return &http.Response{StatusCode: 201}, nil
}

func (frc *FakeRabbitClient) ListPoliciesIn(vhost string) (rec []rabbithole.Policy, err error) {
	return frc.Policies[vhost], nil
}

func (frc *FakeRabbitClient) PutPolicy(vhost string, name string, policy rabbithole.Policy) (res *http.Response, err error) {
	for key, policie := range frc.Policies[vhost] {
		if policie.Name == policy.Name {
			frc.Policies[vhost][key] = policy
			return &http.Response{StatusCode: 200}, nil
		}
	}
	frc.Policies[vhost] = append(frc.Policies[vhost], policy)
	return &http.Response{StatusCode: 201}, nil
}

func (frc *FakeRabbitClient) DeletePolicy(vhost string, name string) (res *http.Response, err error) {
	// If the policy exists or not both api calls returns 204
	for key, policie := range frc.Policies[vhost] {
		if policie.Name == name {
			frc.Policies[vhost][key] = frc.Policies[vhost][len(frc.Policies[vhost])-1]
			frc.Policies[vhost] = frc.Policies[vhost][:len(frc.Policies[vhost])-1]
			return &http.Response{StatusCode: 204}, nil
		}
	}
	return &http.Response{StatusCode: 204}, nil
}

func (frc *FakeRabbitClient) ListPermissionsOf(username string) (rec []rabbithole.PermissionInfo, err error) {
	return frc.Permissions[username], nil
}
func (frc *FakeRabbitClient) UpdatePermissionsIn(vhost, username string, permissions rabbithole.Permissions) (res *http.Response, err error) {
	for key := range frc.Permissions[username] {
		//Update
		if frc.Permissions[username][key].Vhost == vhost {
			frc.Permissions[username][key].Configure = permissions.Configure
			frc.Permissions[username][key].Read = permissions.Read
			frc.Permissions[username][key].Write = permissions.Write
			return &http.Response{StatusCode: 204}, nil
		}
	}
	//Create
	frc.Permissions[username] = append(frc.Permissions[username], rabbithole.PermissionInfo{
		Vhost:     vhost,
		User:      username,
		Configure: permissions.Configure,
		Read:      permissions.Read,
		Write:     permissions.Write,
	})
	return &http.Response{StatusCode: 201}, nil
}
func (frc *FakeRabbitClient) ClearPermissionsIn(vhost, username string) (res *http.Response, err error) {
	for key := range frc.Permissions[username] {
		if frc.Permissions[username][key].Vhost == vhost {
			frc.Permissions[username][key] = frc.Permissions[username][len(frc.Permissions[username])-1]
			frc.Permissions[username] = frc.Permissions[username][:len(frc.Permissions[username])-1]
			return &http.Response{StatusCode: 204}, nil
		}
	}
	return &http.Response{StatusCode: 204}, nil
}
