package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func init() {
	err := apis.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

var k8sclient *client.Client

func getClient() (client.Client, error) {
	if k8sclient != nil {
		return *k8sclient, nil
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg)
	if err != nil {
		return nil, err
	}

	newClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme, Mapper: mapper})
	if err != nil {
		return nil, err
	}
	k8sclient = &newClient
	return *k8sclient, nil
}

func GetObject(ref types.NamespacedName, obj runtime.Object) error {
	fetchClient, err := getClient()
	if err != nil {
		return err
	}

	err = fetchClient.Get(context.Background(), ref, obj)
	if err != nil {
		return err
	}
	return nil
}

func ListSummonPlatform(namespace string) (*summonv1beta1.SummonPlatformList, error) {
	contextClient, err := getClient()
	if err != nil {
		return nil, err
	}
	summonList := &summonv1beta1.SummonPlatformList{}
	listOptions := &client.ListOptions{
		Namespace: namespace,
	}
	err = contextClient.List(context.Background(), listOptions, summonList)
	if err != nil {
		return nil, err
	}
	return summonList, nil
}

func ListObjects(name string, targetList runtime.Object) error {
	tempObj := targetList
	contextClient, err := getClient()
	if err != nil {
		return err
	}
	listOptions := &client.ListOptions{
		Namespace: ParseNamespace(name),
	}
	err = contextClient.List(context.Background(), listOptions, tempObj)
	if err != nil {
		return err
	}

	return nil
}

func GetSummonObject(name string) (*summonv1beta1.SummonPlatform, error) {
	contextClient, err := getClient()
	if err != nil {
		return nil, err
	}
	instance := &summonv1beta1.SummonPlatform{}
	err = contextClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ParseNamespace(name)}, instance)
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func ListNamespaces() ([]string, error) {
	contextClient, err := getClient()
	if err != nil {
		return nil, err
	}
	namespaceList := &corev1.NamespaceList{}
	err = contextClient.List(context.Background(), nil, namespaceList)
	if err != nil {
		return nil, err
	}
	var namespaces []string
	for _, namespace := range namespaceList.Items {
		match := regexp.MustCompile(`summon-*`).MatchString(namespace.Name)
		if match {
			namespaces = append(namespaces, namespace.Name)
		}
	}
	return namespaces, nil
}

func ParseNamespace(instanceName string) string {
	env := strings.Split(instanceName, "-")[1]
	namespace := fmt.Sprintf("summon-%s", env)
	return namespace
}
