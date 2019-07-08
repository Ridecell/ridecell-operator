package actions

import (
	"github.com/Ridecell/ridecell-operator/pkg/webui/kubernetes"
	"github.com/gobuffalo/buffalo"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
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
	instance, err := kubernetes.GetSummonObject(c.Param("instance"))
	if err != nil {
		return err
	}

	// Custom struct to deal with pointer in spec
	type deploymentStruct struct {
		Name              string
		Replicas          int32
		CurrentReplicas   int32
		UpdatedReplicas   int32
		AvailableReplicas int32
	}
	deployments, err := kubernetes.ListDeployments(c.Param("instance"))
	if err != nil {
		return err
	}
	var deploymentList []deploymentStruct
	for _, deployment := range deployments.Items {
		deploymentList = append(deploymentList, deploymentStruct{
			Name:              deployment.Name,
			Replicas:          *deployment.Spec.Replicas,
			CurrentReplicas:   deployment.Status.Replicas,
			UpdatedReplicas:   deployment.Status.UpdatedReplicas,
			AvailableReplicas: deployment.Status.AvailableReplicas,
		})
	}
	c.Set("instance", instance)
	c.Set("deployments", deploymentList)
	return c.Render(200, r.HTML("status/status.html"))
}
