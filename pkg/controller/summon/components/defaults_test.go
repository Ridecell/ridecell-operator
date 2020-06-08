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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

// Note: After autodeploy feature changes, defaults expects either Version or AutoDeploy to be set
// to reconcile.
var _ = Describe("SummonPlatform Defaults Component", func() {
	var comp components.Component

	BeforeEach(func() {
		comp = summoncomponents.NewDefaults()
	})

	It("does nothing on a filled out object", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{
			Hostname:    "foo.example.com",
			Version:     "1.2.3",
			Environment: "dev",
			Replicas: summonv1beta1.ReplicasSpec{
				Web:           intp(2),
				Daphne:        intp(2),
				ChannelWorker: intp(2),
				Static:        intp(2),
			},
		}

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Hostname).To(Equal("foo.example.com"))
		Expect(instance.Spec.Environment).To(Equal("dev"))
		Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.Daphne).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.ChannelWorker).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.Static).To(PointTo(BeEquivalentTo(2)))
	})

	It("sets a default hostname", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Hostname).To(Equal("foo-dev.ridecell.us"))
	})

	It("sets a default prod hostname", func() {
		instance.ObjectMeta.Name = "foo-prod"
		instance.ObjectMeta.Namespace = "summon-prod"
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Hostname).To(Equal("foo-prod.ridecell.com"))
	})

	It("sets a default pull secret", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.PullSecret).To(Equal("pull-secret"))
	})

	It("sets a default dev replicas", func() {
		instance.Namespace = "summon-dev"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Celeryd).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Daphne).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.ChannelWorker).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Static).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.CeleryBeat).To(PointTo(BeEquivalentTo(1)))
	})

	It("sets a default qa replicas", func() {
		instance.Namespace = "summon-qa"
		instance.Spec.Version = "1.2.3"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Celeryd).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Daphne).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.ChannelWorker).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Static).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.CeleryBeat).To(PointTo(BeEquivalentTo(1)))
	})

	It("sets a default uat replicas", func() {
		instance.Namespace = "summon-uat"
		instance.Spec.Version = "1.2.3"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.Celeryd).To(PointTo(BeEquivalentTo(1)))
		Expect(instance.Spec.Replicas.Daphne).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.ChannelWorker).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.Static).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.CeleryBeat).To(PointTo(BeEquivalentTo(1)))
	})

	It("sets a default prod replicas", func() {
		instance.Namespace = "summon-prod"
		instance.Spec.Version = "1.2.3"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(4)))
		Expect(instance.Spec.Replicas.Celeryd).To(PointTo(BeEquivalentTo(4)))
		Expect(instance.Spec.Replicas.Daphne).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.ChannelWorker).To(PointTo(BeEquivalentTo(4)))
		Expect(instance.Spec.Replicas.Static).To(PointTo(BeEquivalentTo(2)))
		Expect(instance.Spec.Replicas.CeleryBeat).To(PointTo(BeEquivalentTo(1)))
	})

	It("sets default component autoscaling to false", func() {
		instance.Spec.Version = "1.2.3"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Replicas.CelerydAuto).To(PointTo(BeEquivalentTo(false)))
	})

	It("allows 0 web replicas", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{
			Version: "1.2.3",
			Replicas: summonv1beta1.ReplicasSpec{
				Web: intp(0),
			},
		}

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(0)))
	})

	It("Sets a default environment with summon prefix", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}
		instance.Namespace = "summon-dev"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Environment).To(Equal("dev"))
	})

	It("Sets a default environment without the summon prefix", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}
		instance.Namespace = "dev"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Environment).To(Equal("dev"))
	})

	It("Set WEB_URL as the first value in aliases", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}
		instance.Spec.Aliases = []string{"xyz.ridecell.com", "abc.ridecell.com"}

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Config["WEB_URL"].String).To(PointTo(Equal("https://xyz.ridecell.com")))
	})

	It("By default enable monitoring for prod Environment", func() {
		instance.Spec = summonv1beta1.SummonPlatformSpec{Version: "1.2.3"}
		instance.Namespace = "prod"
		Expect(comp).To(ReconcileContext(ctx))
		// TEMPORARILY FALSE UNTIL MONITORING IS FIXED
		// Expect(instance.Spec.Monitoring.Enabled).To(PointTo(BeTrue()))
		Expect(instance.Spec.Monitoring.Enabled).To(BeNil())
	})

	Context("with a Redis migration override", func() {
		BeforeEach(func() {
			instance.Spec.MigrationOverrides.RedisHostname = "awsredis"
		})

		It("sets the redis ", func() {
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Spec.Config["ASGI_URL"].String).To(PointTo(Equal("redis://awsredis/1")))
			Expect(instance.Spec.Config["CACHE_URL"].String).To(PointTo(Equal("redis://awsredis/1")))
		})
	})

	It("sets a default prod FIREBASE_APP", func() {
		instance.Namespace = "summon-prod"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Config["FIREBASE_APP"].String).To(PointTo(Equal("ridecell")))
	})

	It("sets a default qa FIREBASE_APP", func() {
		instance.Namespace = "summon-qa"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Config["FIREBASE_APP"].String).To(PointTo(Equal("instant-stage")))
	})

	It("does not set a default FIREBASE_APP if one is already present", func() {
		instance.Namespace = "summon-qa"
		f := "foo"
		instance.Spec.Config = map[string]summonv1beta1.ConfigValue{
			"FIREBASE_APP": summonv1beta1.ConfigValue{String: &f},
		}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Config["FIREBASE_APP"].String).To(PointTo(Equal("foo")))
	})

	It("errors if Spec.Version and Spec.Autodeploy are both set", func() {
		instance.Spec.AutoDeploy = "test-branch"
		_, err := comp.Reconcile(ctx)
		Expect(err).To(HaveOccurred())
	})

	Context("missing Spec.Version", func() {
		It("errors about requiring Spec.Version or Spec.Autodeploy being set", func() {
			instance.Spec.Version = ""
			_, err := comp.Reconcile(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("proceeds to setting defaults if Spec.Autodeploy is set", func() {
			instance.Spec = summonv1beta1.SummonPlatformSpec{
				AutoDeploy: "test-branch",
			}
			instance.Namespace = "summon-dev"

			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Spec.Hostname).To(Equal("foo-dev.ridecell.us"))
			Expect(instance.Spec.PullSecret).To(Equal("pull-secret"))
			Expect(instance.Spec.Replicas.Web).To(PointTo(BeEquivalentTo(1)))
			Expect(instance.Spec.Replicas.Celeryd).To(PointTo(BeEquivalentTo(1)))
			Expect(instance.Spec.Replicas.Daphne).To(PointTo(BeEquivalentTo(1)))
			Expect(instance.Spec.Replicas.ChannelWorker).To(PointTo(BeEquivalentTo(1)))
			Expect(instance.Spec.Replicas.Static).To(PointTo(BeEquivalentTo(1)))
			Expect(instance.Spec.Replicas.CeleryBeat).To(PointTo(BeEquivalentTo(1)))
		})
	})

	It("sets DISPATCH_ENABLED if the dispatch component is enabled", func() {
		instance.Spec.Dispatch.Version = "foo"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Config["DISPATCH_ENABLED"].Bool).To(PointTo(BeTrue()))
		Expect(instance.Spec.Config["DISPATCH_BASE_URL"].String).To(PointTo(Equal("http://foo-dev-dispatch:8000/")))
		Expect(instance.Spec.Dispatch.Resources).ToNot(BeNil())
	})

	It("does not DISPATCH_ENABLED if the dispatch component is not enabled", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Config["DISPATCH_ENABLED"].Bool).To(PointTo(BeFalse()))
		// NOTE: This assertion will be removed when #269 is put back so DISPATCH_BASE_URL is always set.
		Expect(instance.Spec.Config).ToNot(ContainElement("DISPATCH_BASE_URL"))
	})
})
