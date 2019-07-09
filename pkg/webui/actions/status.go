package actions

import (
	"fmt"
	"regexp"

	"github.com/Ridecell/ridecell-operator/pkg/webui/kubernetes"
	"github.com/gobuffalo/buffalo"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// StatusBaseHandler is a default handler to serve up statuses.
func StatusBaseHandler(c buffalo.Context) error {
	namespaces, err := kubernetes.ListNamespaces()
	if err != nil {
		return err
	}
	c.Set("namespaces", namespaces)

	summonMap := map[string][]summonv1beta1.SummonPlatform{}

	for _, namespace := range namespaces {
		summonList, err := kubernetes.ListSummonPlatform(namespace)
		if err != nil {
			return err
		}
		summonMap[namespace] = summonList.Items
	}

	c.Set("summonMap", summonMap)
	return c.Render(200, r.HTML("status/main-status.html"))
}

// StatusHandler is a handler to serve up specific in depth status pages
func StatusHandler(c buffalo.Context) error {
	instanceName := c.Param("instance")
	instance, err := kubernetes.GetSummonObject(instanceName)
	if err != nil {
		return err
	}

	type deploymentStruct struct {
		Name              string
		Replicas          int32
		CurrentReplicas   int32
		UpdatedReplicas   int32
		AvailableReplicas int32
	}
	deployments := &appsv1.DeploymentList{}
	err = kubernetes.ListObjects(instanceName, deployments)
	if err != nil {
		return err
	}
	var deploymentList []deploymentStruct
	for _, deployment := range deployments.Items {
		// Filter out just the relevant objects.
		match := regexp.MustCompile(fmt.Sprintf(`%s-*`, instanceName)).MatchString(deployment.Name)
		if match {
			deploymentList = append(deploymentList, deploymentStruct{
				Name:              deployment.Name,
				Replicas:          *deployment.Spec.Replicas,
				CurrentReplicas:   deployment.Status.Replicas,
				UpdatedReplicas:   deployment.Status.UpdatedReplicas,
				AvailableReplicas: deployment.Status.AvailableReplicas,
			})
		}
	}

	type podStruct struct {
		Name      string
		Readiness string
		Restarts  int32
		Age       string
	}

	pods := &corev1.PodList{}
	err = kubernetes.ListObjects(instanceName, pods)
	if err != nil {
		return err
	}
	var podList []podStruct
	for _, pod := range pods.Items {
		// Filter out just the relevant objects.
		match := regexp.MustCompile(fmt.Sprintf(`%s-*`, instanceName)).MatchString(pod.Name)
		if match {
			var ready int32
			var restarts int32
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Ready {
					ready++
				}
				restarts += containerStatus.RestartCount
			}
			readiness := fmt.Sprintf("%#v/%#v", ready, len(pod.Status.ContainerStatuses))

			podList = append(podList, podStruct{
				Name:      pod.Name,
				Readiness: readiness,
				Restarts:  restarts,
			})
		}
	}

	c.Set("instance", instance)
	c.Set("deployments", deploymentList)
	c.Set("pods", podList)
	return c.Render(200, r.HTML("status/status.html"))
}
