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

package postgresuser

import (
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	postgresusercomponents "github.com/Ridecell/ridecell-operator/pkg/controller/postgresuser/components"
)

// Add creates a new decryptsecrets Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	_, err := components.NewReconciler("postgresuser-controller", mgr, &dbv1beta1.PostgresUser{}, Templates, []components.Component{
		postgresusercomponents.NewDefaults(),
		postgresusercomponents.NewSecret(),
		postgresusercomponents.NewPostgresUser(),
	})
	return err
}
