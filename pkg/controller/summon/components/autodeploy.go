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
	"fmt"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	gcr "github.com/Ridecell/ridecell-operator/pkg/utils/gcr"
)

type AutoDeployComponent struct {
	tagFetcher func()
}

func NewAutoDeploy() *AutoDeployComponent {
	return &AutoDeployComponent{
		tagFetcher: gcr.GetSummonTags,
	}
}

func (c *AutoDeployComponent) InjectMockTagFetcher(tagFetcherFunc func()) {
	c.tagFetcher = tagFetcherFunc
}

func (_ *AutoDeployComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *AutoDeployComponent) WatchChannel() chan event.GenericEvent {
	genericChannel := make(chan event.GenericEvent)
	return genericChannel
}

func (_ *AutoDeployComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	return instance.Spec.AutoDeploy != ""

	/* if instance.Status.Status != "" && instance.Status.Status != summonv1beta1.StatusInitializing && instance.Status.Status != summonv1beta1.StatusDeploying {
		return components.Result{}, nil
	}*/
}

func (comp *AutoDeployComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)
	branchRegex, err := gcr.SanitizeBranchName(instance.Spec.AutoDeploy)

	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to sanitize AutoDeploy: %s for docker image search", instance.Spec.AutoDeploy)
	}

	// Fetch tags from gcr. This triggers cache check and possibly updates tags before assigning version for deployment.
	comp.tagFetcher()
	branchImage, err := gcr.GetLatestImageOfBranch(branchRegex)

	if err != nil {
		return components.Result{}, errors.Wrapf(err, "Failed to find docker image tag for AutoDeploy: %s", instance.Spec.AutoDeploy)
	}

	if branchImage == "" {
		return components.Result{}, errors.Errorf("autodeploy: no matching branch image for %s", instance.Spec.AutoDeploy)
	}

	// Set instance.Spec.Version to trigger and allow Deployment component to handle things
	instance.Spec.Version = branchImage
	fmt.Printf("DEBUG: AUTODEPLOY set version to %s (%s)\n", instance.Spec.Version, branchImage)
	return components.Result{}, nil
}
