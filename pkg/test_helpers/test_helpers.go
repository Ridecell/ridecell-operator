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

package test_helpers

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/onsi/gomega"
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
)

type TestHelpers struct {
	Environment *envtest.Environment
	Cfg         *rest.Config
	Manager     manager.Manager
	ManagerStop chan struct{}
	Client      client.Client
	TestClient  *testClient
	starter     func()
}

type PerTestHelpers struct {
	*TestHelpers
	Namespace         string
	OperatorNamespace string
}

func New() (*TestHelpers, error) {
	helpers := &TestHelpers{}
	_, callerLine, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("Unable to find current filename")
	}
	crdPath := filepath.Join(callerLine, "..", "..", "..", "config", "crds")
	poCrdPath := filepath.Join(callerLine, "..", "..", "..", "vendor", "github.com", "coreos", "prometheus-operator", "example", "prometheus-operator-crd")
	helpers.Environment = &envtest.Environment{
		CRDDirectoryPaths:  []string{crdPath, poCrdPath},
		CRDs:               []*apiextv1beta1.CustomResourceDefinition{postgresv1.PostgresCRD()},
		UseExistingCluster: os.Getenv("USE_EXISTING_CLUSTER") == "true",
	}
	apis.AddToScheme(scheme.Scheme)

	// Initialze the RNG.
	rand.Seed(time.Now().UnixNano())

	return helpers, nil
}

// Start up the test environment. Call from BeforeSuite().
func Start(adder func(manager.Manager) error, cacheClient bool) *TestHelpers {
	helpers, err := New()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Delay the actual startup until the first time SetupTest() is called because
	// when using -focus and -skip, the suite level before is still run even if
	// every test in the suite will be skipped. So for us, it takes a while as it
	// sits around and launches the control plane and then shuts it down for skipped
	// suites.
	helpers.starter = func() {
		// Start the test environment.
		cfg, err := helpers.Environment.Start()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		helpers.Cfg = cfg

		// Create a manager.
		mgr, err := manager.New(helpers.Cfg, manager.Options{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		helpers.Manager = mgr

		// Add the requested controller(s).
		if adder != nil {
			err = adder(helpers.Manager)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}

		// Grab the test client.
		if cacheClient {
			helpers.Client = helpers.Manager.GetClient()
		} else {
			helpers.Client, err = client.New(helpers.Cfg, client.Options{Scheme: helpers.Manager.GetScheme()})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
		helpers.TestClient = &testClient{client: helpers.Client}

		// Start the manager.
		helpers.ManagerStop = make(chan struct{})
		go func() {
			err := mgr.Start(helpers.ManagerStop)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}()

		// Only run the starter the first time.
		helpers.starter = func() {}
	}

	return helpers
}

// Shut down the test environment. Call from AfterSuite().
func (helpers *TestHelpers) Stop() {
	if helpers != nil && helpers.ManagerStop != nil {
		close(helpers.ManagerStop)
	}
	if helpers != nil && helpers.Environment != nil {
		helpers.Environment.Stop()
	}
}

// Set up any needed per test values. Call from BeforeEach().
func (helpers *TestHelpers) SetupTest() *PerTestHelpers {
	helpers.starter()
	newHelpers := &PerTestHelpers{TestHelpers: helpers}

	newHelpers.Namespace = createRandomNamespace(helpers.Client)
	newHelpers.OperatorNamespace = createRandomNamespace(helpers.Client)
	os.Setenv("NAMESPACE", newHelpers.OperatorNamespace)

	return newHelpers
}

// Clean up any per test state. Call from AfterEach().
func (helpers *PerTestHelpers) TeardownTest() {
	err := helpers.Client.Delete(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: helpers.Namespace}})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = helpers.Client.Delete(context.TODO(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: helpers.OperatorNamespace}})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func createRandomNamespace(client client.Client) string {
	namespaceNameBytes := make([]byte, 10)
	rand.Read(namespaceNameBytes)
	namespaceName := "test-" + hex.EncodeToString(namespaceNameBytes)
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespaceName}}
	err := client.Create(context.TODO(), namespace)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return namespaceName
}

// Helper method to make a types.NamespacedName in the correct namespace.
func (h *PerTestHelpers) Name(objName string) types.NamespacedName {
	return types.NamespacedName{Name: objName, Namespace: h.Namespace}
}

// Helper method to show a list of objects, used in AfterEach helpers.
func (h *PerTestHelpers) DebugList(listType kruntime.Object) {
	gvks, unversioned, err := h.Manager.GetScheme().ObjectKinds(listType)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	if unversioned || len(gvks) == 0 {
		fmt.Println("error getting gvks")
		panic("Error getting GVKs")
	}
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvks[0])

	h.TestClient.List(nil, list)
	// TODO Probably replace this whole thing with building maps and yaml.Marshal.
	fmt.Print(gvks[0].Kind[:len(gvks[0].Kind)-4] + ":\n")
	for _, item := range list.Items {
		meta := item.Object["metadata"].(map[string]interface{})
		if meta["namespace"].(string) == h.Namespace {
			status, ok := item.Object["status"].(map[string]interface{})
			if ok {
				fmt.Printf("  %s:\n", meta["name"])
				for key, value := range status {
					fmt.Printf("    %s: %v\n", key, value)
				}
			} else {
				fmt.Printf("  %s: %v\n", meta["name"], item.Object["status"])
			}
		}
	}
}
