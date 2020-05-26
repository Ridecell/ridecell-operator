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

package components_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

var ctx *components.ComponentContext

func TestComponents(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	err := apis.AddToScheme(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ginkgo.RunSpecs(t, "CLOUDAMQP Firewall Components Suite @unit")
}

var _ = ginkgo.BeforeEach(func() {
	// Set up default-y values for tests to use if they want.
	ctx = components.NewTestContext(&corev1.Node{}, nil)
})

func GetTestNodeList() *corev1.NodeList {
	return &corev1.NodeList{
		Items: []corev1.Node{
			&corev1.Node{
				Status: &corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						&corev1.NodeAddress{
							Type:    corev1.NodeExternalIP,
							Address: "1.2.3.4",
						},
					},
				},
			},
		},
	}
}
