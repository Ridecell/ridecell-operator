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

package ridecellingress

import (
	"context"
	"reflect"
	"strings"

	ingressv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/ingress/v1beta1"

	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RidecellIngress Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRidecellIngress{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("ridecellingress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to RidecellIngress
	err = c.Watch(&source.Kind{Type: &ingressv1beta1.RidecellIngress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by RidecellIngress - change this for objects you create
	err = c.Watch(&source.Kind{Type: &v1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &ingressv1beta1.RidecellIngress{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRidecellIngress{}

// ReconcileRidecellIngress reconciles a RidecellIngress object
type ReconcileRidecellIngress struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RidecellIngress object and makes changes based on the state read
// and what is in the RidecellIngress.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions,resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ingress.ridecell.io,resources=ridecellingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.ridecell.io,resources=ridecellingresses/status,verbs=get;update;patch
func (r *ReconcileRidecellIngress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconciler called ", "namespace", "---", "name", "---")
	// Fetch the RidecellIngress instance
	instance := &ingressv1beta1.RidecellIngress{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	//Iterating over each Rule to check hostnames
	for i := 0; i < len(instance.Spec.Rules); i++ {
		hostname := instance.Spec.Rules[i].Host
		//Suffix the hostname with the cluster domain name
		instance.Spec.Rules[i].Host = hostname + "." + populateDomainName(hostname, instance)
		//log.Info("Hostnames ", "iteration", i, " host: ", instance.Spec.Rules[i].Host)
	}

	//Iterating over each TLS to check hostnames
	for i := 0; i < len(instance.Spec.TLS); i++ {
		//Iterating over each host from Host array
		for j := 0; j < len(instance.Spec.TLS[i].Hosts); j++ {
			hostname := instance.Spec.TLS[i].Hosts[j]
			//Suffix the hostname with the cluster domain name
			instance.Spec.TLS[i].Hosts[j] = hostname + "." + populateDomainName(hostname, instance)
			//log.Info("Hostnames ", "iteration", i, " host: ", instance.Spec.TLS[i].Hosts[j])
		}
	}

	//Define required annotations
	if class, ok := instance.Annotations["kubernetes.io/ingress.class"]; !ok || len(class) == 0 {
		instance.Annotations["kubernetes.io/ingress.class"] = "traefik"
	}
	if status, ok := instance.Annotations["kubernetes.io/tls-acme"]; !ok || len(status) == 0 {
		instance.Annotations["kubernetes.io/tls-acme"] = "true"
	}
	if issuer, ok := instance.Annotations["certmanager.k8s.io/cluster-issuer"]; !ok || len(issuer) == 0 {
		instance.Annotations["certmanager.k8s.io/cluster-issuer"] = "letsencrypt-prod"
	}

	// Define the desired Ingress object using RidecellIngress meta and specs
	deploy := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      instance.Labels,
			Annotations: instance.Annotations,
		},
		Spec: instance.Spec,
	}
	if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// TODO(user): Change this for the object type created by your controller
	// Check if the Deployment already exists
	found := &v1beta1.Ingress{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Ingress", "namespace", deploy.Namespace, "name", deploy.Name)
		err = r.Create(context.TODO(), deploy)
		return reconcile.Result{}, err
	} else if err != nil {
		return reconcile.Result{}, err
	}

	//log.Info("After Updating RidecellIngress Status", "namespace", instance.Namespace, "name", instance.Name)
	// TODO(user): Change this for the object type created by your controller
	// Update the found object and write the result back if there are any changes
	if !reflect.DeepEqual(deploy.Spec, found.Spec) {
		found.Spec = deploy.Spec
		log.Info("Updating Ingress", "namespace", deploy.Namespace, "name", deploy.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Get the status of Ingress and copy it to RidecellIngress status
	log.Info("Updating RidecellIngress Status", "namespace", instance.Namespace, "name", instance.Name)
	instance.Status = found.Status
	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func populateDomainName(hostname string, instance *ingressv1beta1.RidecellIngress) string {
	domain := "ridecell.io"
	//Check wheather hostname is a full domain name or not
	if !strings.Contains(hostname, ".") {
		//Populate the cluster domain name according to cloud, region and environment
		if env, ok := instance.Labels["ridecell.io/environment"]; ok || len(env) > 0 {
			domain = env + "." + domain
			if region, ok := instance.Labels["ridecell.io/region"]; ok || len(region) > 0 {
				domain = region + "-" + domain
			}
			if cloud, ok := instance.Labels["ridecell.io/cloud"]; ok || len(cloud) > 0 {
				domain = cloud + "-" + domain
			}
		}
	}
	return domain
}
