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
	"k8s.io/apimachinery/pkg/runtime"
	"os"

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
	instance := ctx.Top.(*dbv1beta1.RDSInstance)

	if instance.Spec.AllocatedStorage == 0 {
		instance.Spec.AllocatedStorage = 100
	}

	if instance.Spec.Engine == "" {
		instance.Spec.Engine = "postgres"
	}

	if instance.Spec.MultiAZ == nil {
		multiAZ := true
		instance.Spec.MultiAZ = &multiAZ
	}

	if instance.Spec.EngineVersion == "" {
		instance.Spec.EngineVersion = "11"
	}

	if instance.Spec.Username == "" {
		instance.Spec.Username = instance.Name
	}

	if instance.Spec.InstanceClass == "" {
		instance.Spec.InstanceClass = "db.t3.micro"
	}

	if instance.Spec.SubnetGroupName == "" {
		instance.Spec.SubnetGroupName = os.Getenv("AWS_SUBNET_GROUP_NAME")
	}

	if instance.Spec.VPCID == "" {
		instance.Spec.VPCID = os.Getenv("VPC_ID")
	}

	if instance.Spec.MaintenanceWindow == "" {
		instance.Spec.MaintenanceWindow = "Sun:07:00-Sun:08:00"
	}

	if instance.Spec.DatabaseName == "" {
		instance.Spec.DatabaseName = instance.Spec.Username
	}

	return components.Result{}, nil
}
