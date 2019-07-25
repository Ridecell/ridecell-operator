/*
Copyright 2019-2020 Ridecell, Inc.

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

package ridecellingress

import (
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ingressv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/ingress/v1beta1"
	ingresscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/ridecellingress/components"
)

// Add creates a new ridecellingress Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	_, err := components.NewReconciler("ridecellingress-controller", mgr, &ingressv1beta1.RidecellIngress{}, Templates, []components.Component{
		ingresscomponents.NewDefaults(),
		ingresscomponents.NewIngress(),
	})
	return err
}
