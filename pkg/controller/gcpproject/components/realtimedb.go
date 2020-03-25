/*
Copyright 2020 Ridecell, Inc.

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

package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"k8s.io/apimachinery/pkg/runtime"

	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

func newRealHTTPClient() (*http.Client, error) {
	client, err := google.DefaultClient(context.Background())
	if err != nil {
		return nil, err
	}
	return client, err
}

type realtimedbComponent struct {
	httpClient *http.Client
	testURL    *string
}

func NewRealtimeDB() *realtimedbComponent {
	var client *http.Client
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		var err error
		client, err = newRealHTTPClient()
		if err != nil {
			// We need better handling of this, so far we haven't have components that can fail to create.
			log.Fatal(err)
		}
	}

	return &realtimedbComponent{httpClient: client}
}

func (comp *realtimedbComponent) InjectHTTPClient(client *http.Client, testURL string) {
	comp.httpClient = client
	comp.testURL = &testURL
}

func (_ *realtimedbComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *realtimedbComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *realtimedbComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*gcpv1beta1.GCPProject)
	// All of this code makes me a bit sad if i'm honest
	if comp.httpClient == nil {
		return components.Result{}, errors.New("gcpproject: firebase credentials not available")
	}

	if instance.Spec.EnableRealtimeDatabase != nil && *instance.Spec.EnableRealtimeDatabase && instance.Spec.EnableFirebase != nil && *instance.Spec.EnableFirebase {
		// Had to use HTTP as the firebase admin package doesn't support the /.settings/rules.json key
		databaseURL := fmt.Sprintf("https://%s.firebaseio.com/.settings/rules.json", instance.Spec.ProjectID)

		// If we're in test mode overwrite datbaseURL to point at mock http server
		if comp.testURL != nil {
			databaseURL = *comp.testURL
		}

		// Parse default rules into interface
		var specRulesJSON interface{}
		err := json.Unmarshal([]byte(instance.Spec.RealtimeDatabaseRules), &specRulesJSON)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "gcpproject: failed to parse spec database rules")
		}

		getResp, err := comp.httpClient.Get(databaseURL)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "gcpproject: failed to get realtimedb rules")
		}
		defer getResp.Body.Close()

		rulesBytes, err := ioutil.ReadAll(getResp.Body)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "gcpproject: failed to read database rules response body")
		}

		var rulesJSON interface{}
		err = json.Unmarshal(rulesBytes, &rulesJSON)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "gcpproject: failed to parse database rules")
		}

		// If received JSON does not match spec sync it
		if !reflect.DeepEqual(rulesJSON, specRulesJSON) {
			// Build a request for updating rules
			req, err := http.NewRequest("PUT", databaseURL, bytes.NewBuffer([]byte(instance.Spec.RealtimeDatabaseRules)))
			if err != nil {
				return components.Result{}, errors.Wrap(err, "gcpproject: failed to create put request for updating database rules")
			}

			putResp, err := comp.httpClient.Do(req)
			if err != nil {
				return components.Result{}, errors.Wrap(err, "gcpproject: failed to put new database rules")
			}
			defer putResp.Body.Close()
		}
	}

	// Clear operation names if exists and move on
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*gcpv1beta1.GCPProject)
		instance.Status.Status = gcpv1beta1.StatusReady
		instance.Status.Message = "Ready"
		return nil
	}}, nil
}
