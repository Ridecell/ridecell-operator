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
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/michaelklishin/rabbit-hole"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

type RabbitMQManager interface {
	ListVhosts() ([]rabbithole.VhostInfo, error)
	PutVhost(string, rabbithole.VhostSettings) (*http.Response, error)
	ListUsers() ([]rabbithole.UserInfo, error)
	PutUser(string, rabbithole.UserSettings) (*http.Response, error)
}

type RabbitMQClientFactory func(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error)

// Implementation of NewTLSClientFactory using rabbithole (i.e. a real client).
func RabbitholeClientFactory(uri string, user string, pass string, t *http.Transport) (RabbitMQManager, error) {
	return rabbithole.NewTLSClient(uri, user, pass, t)
}

// Open a connection to the RabbitMQ server as defined by a RabbitmqConnection object.
func OpenRabbit(_ctx *components.ComponentContext, _dbInfo *dbv1beta1.RabbitmqConnection, clientFactory RabbitMQClientFactory) (RabbitMQManager, error) {
	uri := os.Getenv("RABBITMQ_URI")
	insecure := os.Getenv("RABBITMQ_INSECURE")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure != "",
		},
	}

	parsedUri, err := url.Parse(uri)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse RabbitMQ URL")
	}
	hostUri := url.URL{Scheme: parsedUri.Scheme, Host: parsedUri.Host}
	rmqHost := hostUri.String()
	rmqUser := parsedUri.User.Username()
	rmqPass, _ := parsedUri.User.Password()

	if rmqHost == "" || rmqUser == "" || rmqPass == "" {
		return nil, errors.New("empty rabbitmq connection credentials")
	}

	// Connect to the rabbitmq cluster
	return clientFactory(rmqHost, rmqUser, rmqPass, transport)
}

func RabbitHostAndPort(client RabbitMQManager) (*dbv1beta1.RabbitmqStatusConnection, error) {
	realClient, ok := client.(*rabbithole.Client)
	if ok {
		parsedEndpoint, err := url.Parse(realClient.Endpoint)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse RabbitMQ endpoint URL")
		}
		hostname := parsedEndpoint.Hostname()
		portStr := parsedEndpoint.Port()
		var port int64
		if portStr == "" {
			port = 5671
		} else {
			port, err = strconv.ParseInt(portStr, 10, 16)
			if err != nil {
				return nil, errors.Wrap(err, "unable to parse RabbitMQ endpoint port")
			}
		}
		return &dbv1beta1.RabbitmqStatusConnection{
			Host: hostname,
			Port: int(port),
		}, nil
	} else {
		return &dbv1beta1.RabbitmqStatusConnection{
			Host: "mockhost",
			Port: 5671,
		}, nil
	}
}
