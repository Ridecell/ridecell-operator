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

package summon

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	gcr "github.com/Ridecell/ridecell-operator/pkg/utils/gcr"
)

// Add creates a new Summon Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := components.NewReconciler("summon-platform-controller", mgr, &summonv1beta1.SummonPlatform{}, Templates, []components.Component{
		// Set default values.
		summoncomponents.NewDefaults(),

		// Possibly have Spec.Version value replaced by autodeploy logic.
		summoncomponents.NewAutoDeploy(),

		// Top-level components.
		summoncomponents.NewPullSecret("pullsecret/pullsecret.yml.tpl"),
		summoncomponents.NewPostgres(),

		// aws stuff
		summoncomponents.NewIAMUser("aws/iamuser.yml.tpl"),
		summoncomponents.NewS3Bucket("aws/staticbucket.yml.tpl"),
		summoncomponents.NewMIVS3Bucket("aws/mivbucket.yml.tpl"),

		// GCP stuff.
		summoncomponents.NewServiceAccount(),

		//Rabbitmq components
		summoncomponents.NewRabbitmqVhost("rabbitmq/vhost.yml.tpl"),

		// Secrets components
		summoncomponents.NewSecretKey(),
		summoncomponents.NewFernetRotate(),
		summoncomponents.NewMockCarServerTenant(),
		summoncomponents.NewAppSecret(),
		summoncomponents.NewNewRelic(),

		summoncomponents.NewConfigMap("configmap.yml.tpl"),
		summoncomponents.NewBackup(),
		summoncomponents.NewMigrations("migrations.yml.tpl"),
		summoncomponents.NewMigrateWait(),
		summoncomponents.NewSuperuser(),

		// Redis components.
		summoncomponents.NewPVC("redis/volumeclaim.yml.tpl"),
		summoncomponents.NewRedisDeployment("redis/deployment.yml.tpl"),
		summoncomponents.NewService("redis/service.yml.tpl"),

		// Web components.
		summoncomponents.NewDeployment("web/deployment.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("web/podDisruptionBudget.yml.tpl"),
		summoncomponents.NewService("web/service.yml.tpl"),
		summoncomponents.NewIngress("web/ingress.yml.tpl"),

		// Daphne components.
		summoncomponents.NewDeployment("daphne/deployment.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("daphne/podDisruptionBudget.yml.tpl"),
		summoncomponents.NewService("daphne/service.yml.tpl"),
		summoncomponents.NewIngress("daphne/ingress.yml.tpl"),

		// Static file components.
		summoncomponents.NewDeployment("static/deployment.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("static/podDisruptionBudget.yml.tpl"),
		summoncomponents.NewService("static/service.yml.tpl"),
		summoncomponents.NewIngress("static/ingress.yml.tpl"),

		// Celery components.
		summoncomponents.NewDeploymentWithAutoscaling("celeryd/deployment.yml.tpl", func(s summonv1beta1.SummonPlatform) bool { return s.Spec.Replicas.CelerydAuto }),
		summoncomponents.NewPodDisruptionBudget("celeryd/podDisruptionBudget.yml.tpl"),
		summoncomponents.NewHPA("celeryd/hpa.yml.tpl", func(s summonv1beta1.SummonPlatform) bool { return s.Spec.Replicas.CelerydAuto }),

		// Celerybeat components.
		summoncomponents.NewDeployment("celerybeat/statefulset.yml.tpl"),
		// Does not have a pod disruption budget intentionally
		summoncomponents.NewService("celerybeat/service.yml.tpl"),

		// Channelworker components.
		summoncomponents.NewDeployment("channelworker/deployment.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("channelworker/podDisruptionBudget.yml.tpl"),

		// Dispatch components.
		summoncomponents.NewDeployment("dispatch/deployment.yml.tpl"),
		summoncomponents.NewService("dispatch/service.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("dispatch/podDisruptionBudget.yml.tpl"),

		// Business Portal components.
		summoncomponents.NewDeployment("businessPortal/deployment.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("businessPortal/podDisruptionBudget.yml.tpl"),
		summoncomponents.NewService("businessPortal/service.yml.tpl"),
		summoncomponents.NewIngress("businessPortal/ingress.yml.tpl"),

		// Trip Share components.
		summoncomponents.NewDeployment("tripShare/deployment.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("tripShare/podDisruptionBudget.yml.tpl"),
		summoncomponents.NewService("tripShare/service.yml.tpl"),
		summoncomponents.NewIngress("tripShare/ingress.yml.tpl"),

		// Hw Aux components.
		summoncomponents.NewDeployment("hwAux/deployment.yml.tpl"),
		summoncomponents.NewService("hwAux/service.yml.tpl"),
		summoncomponents.NewPodDisruptionBudget("hwAux/podDisruptionBudget.yml.tpl"),

		// Set Monitoring
		summoncomponents.NewMonitoring(),

		// metrics components
		summoncomponents.NewServiceMonitor("metrics/servicemonitor.yml.tpl"),
		summoncomponents.NewService("metrics/service.yml.tpl"),

		// End of converge status checks.
		summoncomponents.NewStatus(),

		// Notification componenets.
		// Keep Notification at the end of this block
		summoncomponents.NewNotification(),
	})

	if err != nil {
		return err
	}

	gcrChannel := make(chan event.GenericEvent)

	go watchForImages(gcrChannel, c.GetComponentClient())

	err = c.Controller.Watch(
		&source.Channel{Source: gcrChannel},
		&handler.EnqueueRequestForObject{},
	)
	return err
}

// Watches docker image cache for updates and triggers reconciles for summon instances with autodeploy enabled.
func watchForImages(watchChannel chan event.GenericEvent, k8sClient client.Client) {
	for {
		// Sleep at beginning to allow r-o startup and manage autodeploy reconciles for summonplatform using autodeploy.
		time.Sleep(gcr.GetCacheExpiry())

		// Get list of existing SummonPlatforms.
		summonInstances := &summonv1beta1.SummonPlatformList{}
		err := k8sClient.List(context.TODO(), &client.ListOptions{}, summonInstances)
		if err != nil {
			// Make this do something useful or ignore it.
			panic(err)
		}

		// Pick out each that have AutoDeploy enabled and trigger reconcile if cache was updated.
		for n, summonInstance := range summonInstances.Items {
			if summonInstance.Spec.AutoDeploy == "" {
				continue
			}
			watchChannel <- event.GenericEvent{Object: &summonInstances.Items[n], Meta: &summonInstances.Items[n]}
		}
	}
}
