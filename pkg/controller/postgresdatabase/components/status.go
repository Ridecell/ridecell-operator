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

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
)

type statusComponent struct{}

func NewStatus() *statusComponent {
	return &statusComponent{}
}

func (_ *statusComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *statusComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *statusComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*dbv1beta1.PostgresDatabase)
	dbName := instance.Spec.DatabaseName
	status := instance.Status
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*dbv1beta1.PostgresDatabase)
		if status.DatabaseClusterStatus != dbv1beta1.StatusReady && status.DatabaseClusterStatus != postgresv1.ClusterStatusRunning.String() {
			return nil
		}
		if status.DatabaseStatus != dbv1beta1.StatusReady {
			return nil
		}
		if !instance.Spec.SkipUser && status.UserStatus != dbv1beta1.StatusReady {
			return nil
		}
		for ext, _ := range instance.Spec.Extensions {
			status, ok := status.ExtensionStatus[ext]
			if !ok || status != dbv1beta1.StatusReady {
				return nil
			}
		}
		instance.Status.Status = dbv1beta1.StatusReady
		// TODO a better message
		instance.Status.Message = fmt.Sprintf("Database %s ready", dbName)
		return nil
	}}, nil
}
