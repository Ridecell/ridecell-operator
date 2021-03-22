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
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

var fileMode string

const RETRY_INTERVAL = 5 // seconds

// Temporary flag to allow standing up a database prior to migration
var disableDatabase bool

func init() {
	flag.StringVar(&fileMode, "mode", "secret", "switch between secret and config")
	flag.BoolVar(&disableDatabase, "no-db", false, "disable overwriting database config & secrets")
}

func main() {
	flag.Parse()

	// Validate fileMode flag
	if fileMode != "secret" && fileMode != "config" {
		log.Fatal(errors.New(`--mode must be "config" or "secret"`))
	}

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
	args := flag.Args()

	env := args[0]
	serviceName := args[1]

	// Make a fake context.
	ctx := &components.ComponentContext{
		Client:  c,
		Context: context.Background(),
		// A fake root object to match the API of the rest of our code.
		Top: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: serviceName, Namespace: serviceName},
		},
	}

	if fileMode == "secret" {
		// disableDatabase is a temporary flag until all databases are migrated
		if !disableDatabase {
			err = UpdatePostgresSecret(ctx, env, serviceName, c, data)
			if err != nil {
				return err
			}
		}
		err = UpdateRabbitSecret(ctx, env, serviceName, c, data)
		if err != nil {
			return err
		}
		err = UpdateIamuserSecret(ctx, env, serviceName, c, data)
		if err != nil {
			return err
		}
	} else {
		// disableDatabase is a temporary flag until all databases are migrated
		if !disableDatabase {
			err = UpdatePostgresConfig(ctx, env, serviceName, c, data)
			if err != nil {
				return err
			}
		}
		err = UpdateIamuserConfig(ctx, env, serviceName, c, data)
		if err != nil {
			return err
		}

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

func UpdatePostgresConfig(ctx *components.ComponentContext, env string, serviceName string, c client.Client, data map[string]interface{}) error {
	obj, err := getObject(ctx, env, serviceName, "PostgresDatabase")
	if err != nil {
		return err
	}

	pgdb, ok := obj.(*dbv1beta1.PostgresDatabase)
	if !ok {
		return errors.New("Error while converting object into PostgresDatabase")
	}

	pgdbConnection := pgdb.Status.Connection

	_, ok = data["DATABASE"]
	if !ok {
		// Create the key if it doesn't exist
		data["DATABASE"] = map[interface{}]interface{}{}
	}

	dbKey := data["DATABASE"].(map[interface{}]interface{})

	dbKey["HOST"] = pgdbConnection.Host
	dbKey["PORT"] = pgdbConnection.Port
	dbKey["NAME"] = pgdbConnection.Database
	dbKey["USER"] = pgdbConnection.Username

	return nil
}

func UpdateIamuserConfig(ctx *components.ComponentContext, env string, serviceName string, c client.Client, data map[string]interface{}) error {
	obj, err := getObject(ctx, env, serviceName, "Secret")
	if err != nil {
		return err
	}

	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return errors.New("Error while converting object into Secret")
	}

	_, ok = data["AWS"]
	if !ok {
		// Create the key if it doesn't exist
		data["AWS"] = map[interface{}]interface{}{}
	}

	awsKey := data["AWS"].(map[interface{}]interface{})

	awsKey["ACCESS_KEY_ID"] = string(secret.Data["AWS_ACCESS_KEY_ID"])

	return nil
}

func UpdatePostgresSecret(ctx *components.ComponentContext, env string, serviceName string, c client.Client, data map[string]interface{}) error {
	obj, err := getObject(ctx, env, serviceName, "PostgresDatabase")
	if err != nil {
		return err
	}

	pgdb, ok := obj.(*dbv1beta1.PostgresDatabase)
	if !ok {
		return errors.New("Error while converting object into PostgresDatabase")
	}

	pgdbConnection := pgdb.Status.Connection

	// Fetch the postgres database password
	pgdbPassword, err := pgdbConnection.PasswordSecretRef.Resolve(ctx, "password")
	if err != nil {
		return err
	}

	_, ok = data["DATABASE"]
	// Create the key if it doesn't exist
	if !ok {
		data["DATABASE"] = map[interface{}]interface{}{}
	}

	dbKey := data["DATABASE"].(map[interface{}]interface{})

	dbKey["PASSWORD"] = pgdbPassword

	return nil
}

func UpdateRabbitSecret(ctx *components.ComponentContext, env string, serviceName string, c client.Client, data map[string]interface{}) error {
	obj, err := getObject(ctx, env, serviceName, "RabbitmqVhost")
	if err != nil {
		return err
	}

	rmqv, ok := obj.(*dbv1beta1.RabbitmqVhost)
	if !ok {
		return errors.New("Error while converting object into RabbitmqVhost")
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

func UpdateIamuserSecret(ctx *components.ComponentContext, env string, serviceName string, c client.Client, data map[string]interface{}) error {
	obj, err := getObject(ctx, env, serviceName, "Secret")
	if err != nil {
		return err
	}

	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return errors.New("Error while converting object into Secret")
	}

	_, ok = data["AWS"]
	if !ok {
		// Create the key if it doesn't exist
		data["AWS"] = map[interface{}]interface{}{}
	}

	awsKey := data["AWS"].(map[interface{}]interface{})

	awsKey["SECRET_ACCESS_KEY"] = string(secret.Data["AWS_SECRET_ACCESS_KEY"])

	return nil
}

func getObject(ctx *components.ComponentContext, env string, serviceName string, kind string) (*Object, error) {
	var obj runtime.Object
	objName := fmt.Sprintf("svc-%s-%s", env, serviceName)

	switch kind {
	case "RabbitmqVhost":
		obj = &dbv1beta1.RabbitmqVhost{}
	case "PostgresDatabase":
		obj = &dbv1beta1.PostgresDatabase{}
	default:
		obj = &corev1.Secret{}
		objName = fmt.Sprintf("svc-%s-%s.aws-credentials", env, serviceName)
	}

	retry := 2
	var err error
	for retry > 0 {
		err = ctx.Get(ctx.Context, types.NamespacedName{Namespace: serviceName, Name: objName}, obj)
		if err != nil {
			if k8serr.IsNotFound(err) {
				// wait for secrets/configs to be created
				time.Sleep(RETRY_INTERVAL * time.Second)
				retry = retry - 1
				continue
			}
		}
		break
	}
	if k8serr.IsNotFound(err) {
		err = nil
	}
	return obj, err
}
