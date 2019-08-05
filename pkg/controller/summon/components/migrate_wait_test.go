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
	"time"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SummonPlatform Migrations Component", func() {
	comp := summoncomponents.NewMigrateWait()

	Describe("IsReconcilable", func() {
		It("returns false", func() {
			ok := comp.IsReconcilable(ctx)
			Expect(ok).To(BeFalse())
		})

		It("with postmigratewait status", func() {
			instance.Status.Status = summonv1beta1.StatusPostMigrateWait
			Expect(comp).To(ReconcileContext(ctx))
		})
	})

	Context("Reconcile", func() {
		BeforeEach(func() {
			instance.Status.Status = summonv1beta1.StatusPostMigrateWait
		})

		It("with zero value wait", func() {
			instance.Spec.Waits.PostMigrate.Duration = 0
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
			Expect(instance.Status.Wait.Until).To(Equal(""))
		})

		It("with short wait value", func() {
			instance.Spec.Waits.PostMigrate.Duration = time.Second * 5
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusPostMigrateWait))
			parsedTime, err := time.Parse(time.UnixDate, instance.Status.Wait.Until)
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedTime.After(time.Now())).To(BeTrue())

			time.Sleep(time.Second * 6)

			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
			Expect(instance.Status.Wait.Until).To(Equal(""))
		})
	})
})
