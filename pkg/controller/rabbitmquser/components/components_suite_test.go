/*
Copyright 2019-2020 Ridecell, Inc.

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
	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	//corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	//"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var instance *dbv1beta1.RabbitmqUser
var ctx *components.ComponentContext

func TestTemplates(t *testing.T) {
	apis.AddToScheme(scheme.Scheme)
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "RabbitmqUser Components Suite")
}

var _ = ginkgo.BeforeEach(func() {
	// Set up default-y values for tests to use if they want.
	instance = &dbv1beta1.RabbitmqUser{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"},
	}
	//target := &corev1.Secret{
	//	ObjectMeta: metav1.ObjectMeta{Name: "foo.rabbitmq-user-password", Namespace: instance.Namespace},
	//	Data: map[string][]byte{
	//		"password": []byte("rabbitmqpass"),
	//	},
	//}
	ctx = &components.ComponentContext{Top: instance, Client: fake.NewFakeClient(), Scheme: scheme.Scheme}
	//controllerutil.SetControllerReference(instance, target, ctx.Scheme)
	//ctx.Create(ctx.Context, target)

	os.Setenv("RABBITMQ_URI", "https://guest:guest@rabbitmq-prod:5671")
})
