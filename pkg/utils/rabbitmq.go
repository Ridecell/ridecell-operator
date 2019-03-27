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
	"crypto/tls"
	"errors"
	"net/http"
	"os"

	"github.com/michaelklishin/rabbit-hole"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type RabbitMQManager interface {
	ListVhosts() ([]rabbithole.VhostInfo, error)
	PutVhost(string, rabbithole.VhostSettings) (*http.Response, error)
	ListUsers() ([]rabbithole.UserInfo, error)
	PutUser(string, rabbithole.UserSettings) (*http.Response, error)
}

type NewTLSClientFactory func(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error)

// Implementation of NewTLSClientFactory using rabbithole (i.e. a real client).
func RabbitholeTLSClientFactory(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error) {
	return rabbithole.NewTLSClient(uri, user, pass, t)
}

// Open a connection to the RabbitMQ server as defined by a RabbitmqConnection object.
func OpenRabbit(_ctx *components.ComponentContext, dbInfo *dbv1beta1.RabbitmqConnection, clientFactory NewTLSClientFactory) (RabbitMQManager, error) {
	transport := &http.Transport{TLSClientConfig: &tls.Config{
		InsecureSkipVerify: dbInfo.InsecureSkip,
	},
	}

	var rmqHost string
	if dbInfo.Production {
		rmqHost = os.Getenv("RABBITMQ_HOST_PROD")
	} else {
		rmqHost = os.Getenv("RABBITMQ_HOST_DEV")
	}
	rmqUser := os.Getenv("RABBITMQ_SUPERUSER")
	rmqPass := os.Getenv("RABBITMQ_SUPERUSER_PASSWORD")

	if rmqHost == "" || rmqUser == "" || rmqPass == "" {
		return nil, errors.New("empty rabbitmq connection credentials")
	}

	// Connect to the rabbitmq cluster
	return clientFactory(rmqHost, rmqUser, rmqPass, transport)
}
