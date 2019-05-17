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
	"path/filepath"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	flag.Parse()
	apiextv1beta1.AddToScheme(scheme.Scheme)

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

	// Get the list of CRD YAML files.
	if len(os.Args) < 2 {
		log.Fatal("CRD path is required")
	}

	crdFiles, err := filepath.Glob(fmt.Sprintf("%s/*.y*ml", os.Args[1]))
	if err != nil {
		log.Fatal(err)
	}

	// Load the CRDs.
	crds := []*apiextv1beta1.CustomResourceDefinition{}
	for _, crdFile := range crdFiles {
		crdRaw, err := ioutil.ReadFile(crdFile)
		if err != nil {
			log.Fatal(err)
		}
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(crdRaw, nil, nil)
		if err != nil {
			log.Fatal(err)
		}
		crd, ok := obj.(*apiextv1beta1.CustomResourceDefinition)
		if !ok {
			log.Fatalf("File %s is not a CRD", crdFile)
		}
		crds = append(crds, crd)
	}

	// Create or update the CRDs.
	for _, crd := range crds {
		existingCrd := &apiextv1beta1.CustomResourceDefinition{}
		err := c.Get(context.Background(), types.NamespacedName{Name: crd.Name}, existingCrd)
		if errors.IsNotFound(err) {
			log.Printf("Creating %s/%s.%s", crd.Spec.Group, crd.Spec.Version, crd.Spec.Names.Kind)
			err := c.Create(context.Background(), crd)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("Updating %s/%s.%s", crd.Spec.Group, crd.Spec.Version, crd.Spec.Names.Kind)
			existingCrd.Spec = crd.Spec
			err := c.Update(context.Background(), existingCrd)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Printf("Completed")
}
