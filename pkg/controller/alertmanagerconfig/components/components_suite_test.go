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

package components_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"k8s.io/client-go/kubernetes/scheme"

	monitorv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var instance *monitorv1beta1.AlertManagerConfig
var ctx *components.ComponentContext

func TestTemplates(t *testing.T) {
	apis.AddToScheme(scheme.Scheme)
	RegisterFailHandler(Fail)
	RunSpecs(t, "AlertManagerConfig Components Suite @unit")
}

var _ = BeforeEach(func() {
	// Set up default-y values for tests to use if they want.
	instance = &monitorv1beta1.AlertManagerConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "alertmanager-auth", Namespace: "default"},
		Spec: monitorv1beta1.AlertManagerConfigSpec{
			AlertManagerName:      "alertmanager-infra",
			AlertManagerNamespace: "default",
			Data: map[string]string{
				"routes":   "bWF0Y2hfcmU6CiAgc2VydmljZTogXihmb28xfGZvbzJ8YmF6KSQKcmVjZWl2ZXI6IHRlc3QtYWxlcnQKcm91dGVzOgotIG1hdGNoOgogICAgc2V2ZXJpdHk6IGNyaXRpY2FsCnJlY2VpdmVyOiB0ZXN0LWFsZXJ0",
				"receiver": "bmFtZTogJ3Rlc3QtYWxlcnQyJwpzbGFja19jb25maWdzOiAKICAgIC0gc2VuZF9yZXNvbHZlZDogdHJ1ZQo=",
			},
		},
	}
	ctx = components.NewTestContext(instance, nil)
})
