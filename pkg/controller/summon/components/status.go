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

package components

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type statusComponent struct{}

func NewStatus() *statusComponent {
	return &statusComponent{}
}

func (comp *statusComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
	}
}

func (_ *statusComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// Always ready, always waiting ...
	return true
}

func (comp *statusComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	if instance.Status.Status != summonv1beta1.StatusDeploying {
		// If the migrations component didn't already set us to Deploying, don't even bother checking.
		return components.Result{}, nil
	}

	// Grab all (important) Deployments and make sure they are all ready.
	web := &appsv1.Deployment{}
	daphne := &appsv1.Deployment{}
	celeryd := &appsv1.Deployment{}
	channelworker := &appsv1.Deployment{}
	static := &appsv1.Deployment{}
	celerybeat := &appsv1.StatefulSet{}
	kafkaconsumer := &appsv1.Deployment{}

	// Go's lack of generics can fuck right off.
	err := comp.get(ctx, "web", web)
	if err != nil {
		return components.Result{}, err
	}
	err = comp.get(ctx, "daphne", daphne)
	if err != nil {
		return components.Result{}, err
	}
	err = comp.get(ctx, "celeryd", celeryd)
	if err != nil {
		return components.Result{}, err
	}
	err = comp.get(ctx, "channelworker", channelworker)
	if err != nil {
		return components.Result{}, err
	}
	err = comp.get(ctx, "static", static)
	if err != nil {
		return components.Result{}, err
	}
	err = comp.get(ctx, "celerybeat", celerybeat)
	if err != nil {
		return components.Result{}, err
	}
	err = comp.get(ctx, "kafkaconsumer", kafkaconsumer)
	if err != nil {
		return components.Result{}, err
	}

	if os.Getenv("ENABLE_NEW_STATUS_CHECK") == "true" {
		dispatch := &appsv1.Deployment{}
		businessPortal := &appsv1.Deployment{}
		tripShare := &appsv1.Deployment{}
		hwAux := &appsv1.Deployment{}

		err = comp.get(ctx, "dispatch", dispatch)
		if err != nil {
			return components.Result{}, err
		}
		err = comp.get(ctx, "businessportal", businessPortal)
		if err != nil {
			return components.Result{}, err
		}
		err = comp.get(ctx, "tripshare", tripShare)
		if err != nil {
			return components.Result{}, err
		}
		err = comp.get(ctx, "hwaux", hwAux)
		if err != nil {
			return components.Result{}, err
		}

		// the bigger newer check
		if comp.isReady(web) && comp.isReady(daphne) &&
			comp.isReady(celeryd) && comp.isReady(channelworker) &&
			comp.isReady(static) && comp.isReady(celerybeat) &&
			comp.isReady(dispatch) && comp.isReady(businessPortal) &&
			comp.isReady(tripShare) && comp.isReady(hwAux) && comp.isReady(kafkaconsumer) {
			return components.Result{StatusModifier: func(obj runtime.Object) error {
				instance := obj.(*summonv1beta1.SummonPlatform)
				instance.Status.Status = summonv1beta1.StatusReady
				instance.Status.Message = fmt.Sprintf("Cluster %s ready", instance.Name)
				return nil
			}}, nil
		}
		return components.Result{}, nil
	}

	// The big check!
	if web.Spec.Replicas != nil && web.Status.AvailableReplicas == *web.Spec.Replicas &&
		daphne.Spec.Replicas != nil && daphne.Status.AvailableReplicas == *daphne.Spec.Replicas &&
		celeryd.Spec.Replicas != nil && celeryd.Status.AvailableReplicas == *celeryd.Spec.Replicas &&
		channelworker.Spec.Replicas != nil && channelworker.Status.AvailableReplicas == *channelworker.Spec.Replicas &&
		static.Spec.Replicas != nil && static.Status.AvailableReplicas == *static.Spec.Replicas &&
		kafkaconsumer.Spec.Replicas != nil && kafkaconsumer.Status.AvailableReplicas == *kafkaconsumer.Spec.AvailableReplicas &&
		// Note this one is different, available vs ready.
		celerybeat.Spec.Replicas != nil && celerybeat.Status.ReadyReplicas == *celerybeat.Spec.Replicas {
		// TODO: Add an actual HTTP self check in here.
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Status = summonv1beta1.StatusReady
			instance.Status.Message = fmt.Sprintf("Cluster %s ready", instance.Name)
			return nil
		}}, nil
	}

	// Not ready, alas.
	return components.Result{}, nil
}

// Short helper because we need to do this 6 times.
func (comp *statusComponent) get(ctx *components.ComponentContext, part string, obj runtime.Object) error {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	name := types.NamespacedName{Name: fmt.Sprintf("%s-%s", instance.Name, part), Namespace: instance.Namespace}
	err := ctx.Get(ctx.Context, name, obj)
	// If it's a NotFound error, just ignore it since we don't want to that to fail things and the zero value will fail later on.
	if err != nil && !kerrors.IsNotFound(err) {
		return errors.Wrapf(err, "status: unable to get Deployment or StatefulSet %s for %s subsystem", name, part)
	}
	return nil
}

func (comp *statusComponent) isReady(robject runtime.Object) bool {
	statefulset, ok := robject.(*appsv1.StatefulSet)
	if ok {
		if statefulset.Spec.Replicas != nil && statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas && statefulset.Status.UpdatedReplicas == *statefulset.Spec.Replicas {
			return true
		}
		return false
	}

	// if it's neither thing panic
	deployment := robject.(*appsv1.Deployment)
	if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas && deployment.Status.ReadyReplicas == *deployment.Spec.Replicas && deployment.Status.UnavailableReplicas == 0 {
		return true
	}
	return false
}
