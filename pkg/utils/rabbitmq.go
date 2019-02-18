/*
Copyright 2018-2019 Ridecell, Inc.

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

package utils

import (
	"github.com/michaelklishin/rabbit-hole"
	"net/http"
)

type RabbitMQManager interface {
	ListVhosts() ([]rabbithole.VhostInfo, error)
	PutVhost(string, rabbithole.VhostSettings) (*http.Response, error)
	ListUsers() ([]rabbithole.UserInfo, error)
	PutUser(string, rabbithole.UserSettings) (*http.Response, error)
}

type NewTLSClientFactory func(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error)
