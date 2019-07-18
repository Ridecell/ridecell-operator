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

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

func main() {
	flag.Parse()

	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal(err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		log.Fatal(err)
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		log.Fatal(err)
	}

	err = Run(c)
	if err != nil {
		log.Fatal(err)
	}
}

func Run(c client.Client) error {
	// Load the YAML from stdin.
	data, err := LoadYAML()
	if err != nil {
		return err
	}

	// Process the YAML.
	env := os.Args[1]
	serviceName := os.Args[2]
	err = Update(env, serviceName, c, data)
	if err != nil {
		return err
	}

	// Serialize the updated YAML to stdout.
	err = DumpYAML(data)
	if err != nil {
		return err
	}

	return nil
}

func LoadYAML() (map[string]interface{}, error) {
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{}
	err = yaml.Unmarshal(bytes, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func DumpYAML(data map[string]interface{}) error {
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

func Update(env, serviceName string, c client.Client, data map[string]interface{}) error {
	// Make a fake context.
	ctx := &components.ComponentContext{
		Client:  c,
		Context: context.Background(),
		// A fake root object to match the API of the rest of our code.
		Top: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: serviceName},
		},
	}

	// Fetch the RabbitmqVhost object.
	rmqv := &dbv1beta1.RabbitmqVhost{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Namespace: serviceName, Name: fmt.Sprintf("svc-%s-%s", env, serviceName)}, rmqv)
	if err != nil {
		return err
	}
	rabbitmqConnection := rmqv.Status.Connection

	// Fetch the password.
	rabbitmqPassword, err := rabbitmqConnection.PasswordSecretRef.Resolve(ctx, "password")
	if err != nil {
		return err
	}

	// Create the CELERY_BROKER_URL.
	data["CELERY_BROKER_URL"] = fmt.Sprintf("pyamqp://%s:%s@%s/%s?ssl=true", rabbitmqConnection.Username, rabbitmqPassword, rabbitmqConnection.Host, rabbitmqConnection.Vhost)

	return nil
}
