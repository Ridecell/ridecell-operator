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
	"k8s.io/apimachinery/pkg/runtime"
	"regex"
	"time"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type defaultsComponent struct {
}

func NewDefaults() *defaultsComponent {
	return &defaultsComponent{}
}

func (_ *defaultsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *defaultsComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *defaultsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.RDSSnapshot)

	if instance.Spec.SnapshotID == "" {
		// Add 0 seconds to convert creationTimestamp from metav1.Timestamp to time.Time
		creationTimestamp := instance.ObjectMeta.CreationTimestamp.Add(time.Second * 0)
		// Reformat the timestamp to be friendly with rds snapshot naming restrictions
		curTimeString := time.Time.Format(creationTimestamp, CustomTimeLayout)
		instance.Spec.SnapshotID = fmt.Sprintf("%s-%s", instance.Name, curTimeString)
	}

	// sanitize snapshot id, replace any special chars with `-`, also remove consecutive `-`
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		return components.Result{}, err
	}
	instance.Spec.SnapshotID = reg.ReplaceAllString(instance.Spec.SnapshotID, "-")

	return components.Result{}, nil
}
