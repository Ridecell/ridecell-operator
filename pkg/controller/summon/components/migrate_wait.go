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

package components

import (
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type migrateWaitComponent struct {
}

func NewMigrateWait() *migrateWaitComponent {
	return &migrateWaitComponent{}
}

func (comp *migrateWaitComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *migrateWaitComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	// We aren't waiting, no need to run
	if instance.Status.Status != summonv1beta1.StatusPostMigrateWait {
		return false
	}
	return true
}

func (comp *migrateWaitComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	var waitUntil time.Time
	waitUntilString := instance.Status.Wait.Until

	if waitUntilString == "" {
		waitUntil = time.Now().Add(instance.Spec.Waits.PostMigrate.Duration)
	} else {
		parsedTime, err := time.Parse(time.UnixDate, waitUntilString)
		if err != nil {
			return components.Result{}, errors.Wrap(err, "migrate_wait: failed to parse time stats")
		}
		waitUntil = parsedTime
	}

	if !metav1.Now().After(waitUntil) && instance.Spec.Waits.PostMigrate.Duration != 0 {
		return components.Result{
			StatusModifier: func(obj runtime.Object) error {
				instance := obj.(*summonv1beta1.SummonPlatform)
				instance.Status.Wait.Until = waitUntil.Format(time.UnixDate)
				return nil
			},
			RequeueAfter: instance.Spec.Waits.PostMigrate.Duration,
		}, nil
	}

	// No longer waiting, set status to deploying to continue
	return components.Result{
		StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta1.SummonPlatform)
			instance.Status.Status = summonv1beta1.StatusDeploying
			// Set wait time to zero value
			instance.Status.Wait.Until = ""
			return nil
		},
	}, nil
}
