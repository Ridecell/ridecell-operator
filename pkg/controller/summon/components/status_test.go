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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("SummonPlatform Status Component", func() {
	var webDeployment *appsv1.Deployment
	var daphneDeployment *appsv1.Deployment
	var celerydDeployment *appsv1.Deployment
	var channelworkersDeployment *appsv1.Deployment
	var staticDeployment *appsv1.Deployment
	var dispatchDeployment *appsv1.Deployment
	var businessPortalDeployment *appsv1.Deployment
	var tripShareDeployment *appsv1.Deployment
	var hwAuxDeployment *appsv1.Deployment
	var celerybeatStatefulSet *appsv1.StatefulSet
	var kafkaconsumerDeployment *appsv1.Deployment
	makeClient := func() client.Client {
		return fake.NewFakeClient(instance, webDeployment, daphneDeployment, celerydDeployment,
			channelworkersDeployment, staticDeployment, celerybeatStatefulSet, kafkaconsumerDeployment)
	}

	BeforeEach(func() {
		os.Setenv("ENABLE_NEW_STATUS_CHECK", "false")
		webDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-web", Namespace: "summon-dev"},
			Spec: appsv1.DeploymentSpec{
				Replicas: intp(2),
			},
		}
		daphneDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-daphne", Namespace: "summon-dev"},
			Spec: appsv1.DeploymentSpec{
				Replicas: intp(2),
			},
		}
		celerydDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-celeryd", Namespace: "summon-dev"},
			Spec: appsv1.DeploymentSpec{
				Replicas: intp(2),
			},
		}
		kafkaconsumerdDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-kafkaconsumer", Namespace: "summon-dev"},
			Spec: appsv1.DeploymentSpec{
				Replicas: intp(2),
			},
		}
		channelworkersDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-channelworker", Namespace: "summon-dev"},
			Spec: appsv1.DeploymentSpec{
				Replicas: intp(2),
			},
		}
		staticDeployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-static", Namespace: "summon-dev"},
			Spec: appsv1.DeploymentSpec{
				Replicas: intp(2),
			},
		}
		celerybeatStatefulSet = &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-celerybeat", Namespace: "summon-dev"},
			Spec: appsv1.StatefulSetSpec{
				Replicas: intp(2),
			},
		}

		ctx.Client = makeClient()
	})

	It("watches 2 types", func() {
		comp := summoncomponents.NewStatus()
		Expect(comp.WatchTypes()).To(HaveLen(2))
	})

	It("is always reconciable", func() {
		comp := summoncomponents.NewStatus()
		Expect(comp.IsReconcilable(ctx)).To(BeTrue())
	})

	It("sets the status to ready", func() {
		webDeployment.Status.AvailableReplicas = 2
		daphneDeployment.Status.AvailableReplicas = 2
		celerydDeployment.Status.AvailableReplicas = 2
		channelworkersDeployment.Status.AvailableReplicas = 2
		staticDeployment.Status.AvailableReplicas = 2
		celerybeatStatefulSet.Status.ReadyReplicas = 2
		kafkaconsumerDeployment.Status.AvailableReplicas = 2
		instance.Status.Status = summonv1beta1.StatusDeploying
		ctx.Client = makeClient()

		comp := summoncomponents.NewStatus()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusReady))
	})

	It("doesn't update if still migrating", func() {
		instance.Status.Status = summonv1beta1.StatusMigrating

		comp := summoncomponents.NewStatus()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusMigrating))
	})

	It("doesn't update if deployments aren't ready", func() {
		instance.Status.Status = summonv1beta1.StatusDeploying

		comp := summoncomponents.NewStatus()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
	})

	It("doesn't update if deployments are partially ready", func() {
		webDeployment.Status.AvailableReplicas = 1
		instance.Status.Status = summonv1beta1.StatusDeploying
		ctx.Client = makeClient()

		comp := summoncomponents.NewStatus()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
	})

	It("doesn't update if deployments don't exist yet", func() {
		instance.Status.Status = summonv1beta1.StatusDeploying
		ctx.Client = fake.NewFakeClient()

		comp := summoncomponents.NewStatus()
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
	})

	Context("new status check behavior", func() {
		BeforeEach(func() {
			os.Setenv("ENABLE_NEW_STATUS_CHECK", "true")
			dispatchDeployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-dispatch", Namespace: "summon-dev"},
				Spec: appsv1.DeploymentSpec{
					Replicas: intp(2),
				},
			}
			businessPortalDeployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-businessportal", Namespace: "summon-dev"},
				Spec: appsv1.DeploymentSpec{
					Replicas: intp(2),
				},
			}
			tripShareDeployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-tripshare", Namespace: "summon-dev"},
				Spec: appsv1.DeploymentSpec{
					Replicas: intp(2),
				},
			}
			hwAuxDeployment = &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "foo-dev-hwaux", Namespace: "summon-dev"},
				Spec: appsv1.DeploymentSpec{
					Replicas: intp(2),
				},
			}
		})
		It("sets the status to ready", func() {
			updateDeploymentStatus := func(deployment *appsv1.Deployment) {
				deployment.Status.ReadyReplicas = 2
				deployment.Status.UpdatedReplicas = 2
			}
			updateDeploymentStatus(webDeployment)
			updateDeploymentStatus(daphneDeployment)
			updateDeploymentStatus(celerydDeployment)
			updateDeploymentStatus(channelworkersDeployment)
			updateDeploymentStatus(staticDeployment)
			updateDeploymentStatus(dispatchDeployment)
			updateDeploymentStatus(businessPortalDeployment)
			updateDeploymentStatus(tripShareDeployment)
			updateDeploymentStatus(hwAuxDeployment)
			updateDeploymentStatus(kafkaconsumerDeployment)
			celerybeatStatefulSet.Status.ReadyReplicas = 2
			celerybeatStatefulSet.Status.UpdatedReplicas = 2

			instance.Status.Status = summonv1beta1.StatusDeploying
			ctx.Client = fake.NewFakeClient(instance, webDeployment, daphneDeployment, celerydDeployment,
				channelworkersDeployment, staticDeployment, celerybeatStatefulSet, dispatchDeployment,
				businessPortalDeployment, tripShareDeployment, hwAuxDeployment, kafkaconsumerDeployment)

			comp := summoncomponents.NewStatus()
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusReady))
		})

		It("doesn't update if still migrating", func() {
			instance.Status.Status = summonv1beta1.StatusMigrating

			comp := summoncomponents.NewStatus()
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusMigrating))
		})

		It("doesn't update if deployments aren't ready", func() {
			instance.Status.Status = summonv1beta1.StatusDeploying

			comp := summoncomponents.NewStatus()
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
		})

		It("doesn't update if deployments are partially ready", func() {
			webDeployment.Status.AvailableReplicas = 1
			instance.Status.Status = summonv1beta1.StatusDeploying
			ctx.Client = fake.NewFakeClient(instance, webDeployment, daphneDeployment, celerydDeployment,
				channelworkersDeployment, staticDeployment, celerybeatStatefulSet, dispatchDeployment,
				businessPortalDeployment, tripShareDeployment, hwAuxDeployment)

			comp := summoncomponents.NewStatus()
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
		})

		It("doesn't update if deployments don't exist yet", func() {
			instance.Status.Status = summonv1beta1.StatusDeploying
			ctx.Client = fake.NewFakeClient()

			comp := summoncomponents.NewStatus()
			Expect(comp).To(ReconcileContext(ctx))
			Expect(instance.Status.Status).To(Equal(summonv1beta1.StatusDeploying))
		})
	})
})
