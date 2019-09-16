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

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	gcpv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/gcp/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/controller/serviceaccount"
)

var instance *gcpv1beta1.GCPServiceAccount
var ctx *components.ComponentContext

func TestComponents(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	err := apis.AddToScheme(scheme.Scheme)
        gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ginkgo.RunSpecs(t, "serviceaccount Components Suite @unit")
}

var _ = ginkgo.BeforeEach(func() {
	// Set up default-y values for tests to use if they want.
	instance = &gcpv1beta1.GCPServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "test-user", Namespace: "default"},
		Spec: gcpv1beta1.GCPServiceAccountSpec{
			AccountName: "test-user",
			Project:     "test-project",
		},
	}
	ctx = components.NewTestContext(instance, serviceaccount.Templates)
})
