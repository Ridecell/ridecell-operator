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
	"os"
	"testing"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/controller/monitor"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var instance *monitoringv1beta1.Monitor
var ctx *components.ComponentContext

func TestComponents(t *testing.T) {
	apis.AddToScheme(scheme.Scheme)
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Monitor Components Suite @unit")
}

var _ = ginkgo.BeforeEach(func() {
	os.Setenv("PG_ROUTING_KEY", "ALLISWELL")
	// Set up default-y values for tests to use if they want.
	instance = &monitoringv1beta1.Monitor{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
	}
	ctx = components.NewTestContext(instance, monitor.Templates)
})
