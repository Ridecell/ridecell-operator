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
	"context"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	promv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

var _ = Describe("servicemonitor Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = summoncomponents.NewServiceMonitor("web/servicemonitor.yml.tpl")
	})

	It("runs with nil flag", func() {
		Expect(comp).To(ReconcileContext(ctx))

		monitor := &promv1.ServiceMonitor{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, monitor)
		Expect(kerrors.IsNotFound(err)).To(BeTrue())
	})

	It("runs with false flag", func() {
		falseFlag := false
		instance.Spec.Metrics.Web = &falseFlag
		Expect(comp).To(ReconcileContext(ctx))

		monitor := &promv1.ServiceMonitor{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, monitor)
		Expect(kerrors.IsNotFound(err)).To(BeTrue())
	})

	It("runs with true flag", func() {
		trueBool := true
		instance.Spec.Metrics.Web = &trueBool
		Expect(comp).To(ReconcileContext(ctx))

		monitor := &promv1.ServiceMonitor{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, monitor)
		Expect(err).ToNot(HaveOccurred())
	})

	It("delete servicemonitor after disable", func() {
		trueBool := true
		instance.Spec.Metrics.Web = &trueBool
		Expect(comp).To(ReconcileContext(ctx))

		monitor := &promv1.ServiceMonitor{}
		err := ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, monitor)
		Expect(err).ToNot(HaveOccurred())

		instance.Spec.Metrics.Web = nil
		Expect(comp).To(ReconcileContext(ctx))
		err = ctx.Client.Get(context.TODO(), types.NamespacedName{Name: "foo-dev-web", Namespace: instance.Namespace}, monitor)
		Expect(kerrors.IsNotFound(err)).To(BeTrue())
	})
})
