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

package components_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/controller/summon"
)

var instance *summonv1beta1.SummonPlatform
var ctx *components.ComponentContext

func TestComponents(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	err := apis.AddToScheme(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ginkgo.RunSpecs(t, "SummonPlatform Components Suite @unit")
}

var _ = ginkgo.BeforeEach(func() {
	// Set up default-y values for tests to use if they want.
	instance = &summonv1beta1.SummonPlatform{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-dev", Namespace: "summon-dev"},
		Spec: summonv1beta1.SummonPlatformSpec{
			Hostname: "foo.ridecell.us",
			Version:  "1.2.3",
		},
		Status: summonv1beta1.SummonPlatformStatus{
			Notification: summonv1beta1.NotificationStatus{NotifyVersion: "1.2.3"}},
	}
	instance.Spec.Secrets = []string{"testsecret"}
	ctx = components.NewTestContext(instance, summon.Templates)
})

// Return an int pointer because &1 doesn't work in Go.
func intp(n int32) *int32 {
	return &n
}
