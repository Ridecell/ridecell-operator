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

package components

import (
	ingressv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/ingress/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

type ridecellingressComponent struct{}

func NewIngress() *ridecellingressComponent {
	return &ridecellingressComponent{}
}

func (comp *ridecellingressComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&extv1beta1.Ingress{},
	}
}

func (_ *ridecellingressComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *ridecellingressComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	// Fetch the RidecellIngress instance
	instance := ctx.Top.(*ingressv1beta1.RidecellIngress)
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, instance)
	if err != nil {
		// Error reading the object - requeue the request.
		return components.Result{}, errors.Wrapf(err, "instance of ridecellingress not found")
	}

	// If no annotations provided, create one
	if instance.Annotations == nil {
		instance.Annotations = map[string]string{}
	}
	// Fill in defaults for Annotations
	if class, ok := instance.Annotations["kubernetes.io/ingress.class"]; !ok || len(class) == 0 {
		instance.Annotations["kubernetes.io/ingress.class"] = ingressv1beta1.IngressClass
	}
	if status, ok := instance.Annotations["kubernetes.io/tls-acme"]; !ok || len(status) == 0 {
		instance.Annotations["kubernetes.io/tls-acme"] = ingressv1beta1.TLS_ACME
	}
	if issuer, ok := instance.Annotations["certmanager.k8s.io/cluster-issuer"]; !ok || len(issuer) == 0 {
		instance.Annotations["certmanager.k8s.io/cluster-issuer"] = ingressv1beta1.ClusterIssuer
	}

	//Iterating over each Rule to check hostnames
	for i := 0; i < len(instance.Spec.Rules); i++ {
		hostname := instance.Spec.Rules[i].Host
		//Check wheather hostname is a full domain name or not
		if !strings.Contains(hostname, ".") {
			//Suffix the hostname with the cluster domain name
			clusterDomain, err := populateDomainName(instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "Domain error")
			}
			instance.Spec.Rules[i].Host = hostname + "." + clusterDomain
		}
	}

	//Iterating over each TLS to check hostnames
	for i := 0; i < len(instance.Spec.TLS); i++ {
		//Iterating over each host from Host array
		for j := 0; j < len(instance.Spec.TLS[i].Hosts); j++ {
			hostname := instance.Spec.TLS[i].Hosts[j]
			//Check wheather hostname is a full domain name or not
			if !strings.Contains(hostname, ".") {
				//Suffix the hostname with the cluster domain name
				clusterDomain, err := populateDomainName(instance)
				if err != nil {
					return components.Result{}, errors.Wrapf(err, "Domain error")
				}
				instance.Spec.TLS[i].Hosts[j] = hostname + "." + clusterDomain
			}
		}
	}

	res, _, err := ctx.CreateOrUpdate("ingress.yml.tpl", nil, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*extv1beta1.Ingress)
		existing := existingObj.(*extv1beta1.Ingress)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		existing.Annotations = goal.Annotations
		existing.Labels = goal.Labels
		return nil
	})
	if err != nil {
		return res, errors.Wrapf(err, "Ingress deployment: failed")
	}
	// Mark status as Success
	res.StatusModifier = func(obj runtime.Object) error {
		instance := obj.(*ingressv1beta1.RidecellIngress)
		ingressInstance := &extv1beta1.Ingress{}
		instance.Status.Status = "Success"
		instance.Status.Message = "Ingress created."
		instance.Status.IngressStatus = ingressInstance.Status
		return nil
	}
	return res, nil
}

func populateDomainName(instance *ingressv1beta1.RidecellIngress) (string, error) {
	domain := ingressv1beta1.RootDomain
	env, ok_env := instance.Labels["ridecell.io/environment"]
	region, ok_reg := instance.Labels["ridecell.io/region"]

	if !(ok_env && ok_reg) {
		return "", errors.Errorf("ridecell.io/environment and ridecell.io/region labels are required.")
	}
	//Populate the cluster domain name according to cloud, region and environment
	domain = region + "-" + env + "." + domain
	if cloud, ok := instance.Labels["ridecell.io/cloud"]; ok || len(cloud) > 0 {
		domain = cloud + "-" + domain
	}
	return domain, nil
}
