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

package rdssnapshot

import (
	"context"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	dbv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/db/v1beta1"
	rdssnapshotcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rdssnapshot/components"
)

// Add creates a new rds snapshot Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	c, err := components.NewReconciler("rds-snapshot-controller", mgr, &dbv1beta1.RDSSnapshot{}, nil, []components.Component{
		rdssnapshotcomponents.NewDefaults(),
		rdssnapshotcomponents.NewRDSSnapshot(),
	})
	if err != nil {
		return err
	}

	genericChannel := make(chan event.GenericEvent)

	go watchTTL(genericChannel, c.GetComponentClient())

	err = c.Controller.Watch(
		&source.Channel{Source: genericChannel},
		&handler.EnqueueRequestForObject{},
	)
	return err
}

func watchTTL(watchChannel chan event.GenericEvent, k8sClient client.Client) {
	rdsSnapshots := &dbv1beta1.RDSSnapshotList{}
	err := k8sClient.List(context.TODO(), &client.ListOptions{}, rdsSnapshots)
	if err != nil {
		// Make this do something useful or ignore it.
		panic(err)
	}

	for _, rdsSnapshot := range rdsSnapshots.Items {
		// ignore object early if object has no TTL set
		if rdsSnapshot.Spec.TTL == 0 {
			continue
		}

		// Check if our object is expired
		deletionTime := rdsSnapshot.ObjectMeta.CreationTimestamp.Add(rdsSnapshot.Spec.TTL)
		if time.Now().After(deletionTime) {
			// Send a generic event to our watched channel to cause a reconcile of specified object
			watchChannel <- event.GenericEvent{Object: &rdsSnapshot, Meta: &rdsSnapshot}
		}
	}
	time.Sleep(time.Minute)
}
