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

package test_helpers

import (
	"context"
	"reflect"
	"time"

	"github.com/onsi/gomega"
	gtypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The default timeout for EventuallyGet().
var DefaultTimeout = 30 * time.Second

// Implementation to match controller-runtime's client.Client interface.
type testClient struct {
	client client.Client
}

type testStatusClient struct {
	client client.StatusWriter
}

func (c *testClient) Get(key client.ObjectKey, obj runtime.Object) {
	err := c.client.Get(context.Background(), key, obj)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
}

func (c *testClient) List(obj runtime.Object, opts ...client.ListOptionFunc) {
	err := c.client.List(context.Background(), obj, opts...)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
}

func (c *testClient) Create(obj runtime.Object) {
	err := c.client.Create(context.Background(), obj)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
}

func (c *testClient) Delete(obj runtime.Object, opts ...client.DeleteOptionFunc) {
	err := c.client.Delete(context.Background(), obj, opts...)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
}

func (c *testClient) Update(obj runtime.Object) {
	err := c.client.Update(context.Background(), obj)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
}

// Implementation to match StatusClient.
func (c *testClient) Status() *testStatusClient {
	return &testStatusClient{client: c.client.Status()}
}

func (c *testStatusClient) Update(obj runtime.Object) {
	err := c.client.Update(context.Background(), obj)
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
}

// Flexible helper, mostly used for waiting for an object to be available.
type eventuallyGetOptions struct {
	timeout     time.Duration
	valueGetter EventuallyGetValueGetter
	matcher     gtypes.GomegaMatcher
}

type eventuallyGetOptionsSetter func(*eventuallyGetOptions)
type EventuallyGetValueGetter func(runtime.Object) (interface{}, error)

// Set the timeout to a non-default value for EventuallyGet().
func (_ *testClient) EventuallyTimeout(timeout time.Duration) eventuallyGetOptionsSetter {
	return func(o *eventuallyGetOptions) {
		o.timeout = timeout
	}
}

// Set a value getter, to poll until the requested value matches.
func (_ *testClient) EventuallyValue(matcher gtypes.GomegaMatcher, getter EventuallyGetValueGetter) eventuallyGetOptionsSetter {
	return func(o *eventuallyGetOptions) {
		o.matcher = matcher
		o.valueGetter = getter
	}
}

// A common case of a value getter for obj.Status.Status.
func (c *testClient) EventuallyStatus(status string) eventuallyGetOptionsSetter {
	return c.EventuallyValue(gomega.Equal(status), func(obj runtime.Object) (interface{}, error) {
		// Yes using reflect is kind of gross but it's test-only code so meh.
		return reflect.ValueOf(obj).Elem().FieldByName("Status").FieldByName("Status").String(), nil
	})
}

// Like a normal Get but run in a loop. By default it will wait until the call succeeds, but can optionally wait for a value.
func (c *testClient) EventuallyGet(key client.ObjectKey, obj runtime.Object, optSetters ...eventuallyGetOptionsSetter) {
	opts := eventuallyGetOptions{timeout: DefaultTimeout}
	for _, optSetter := range optSetters {
		optSetter(&opts)
	}

	if opts.valueGetter != nil {
		gomega.EventuallyWithOffset(1, func() (interface{}, error) {
			var value interface{}
			err := c.client.Get(context.Background(), key, obj)
			if err == nil {
				value, err = opts.valueGetter(obj)
			}
			return value, err
		}, opts.timeout).Should(opts.matcher)
	} else {
		gomega.EventuallyWithOffset(1, func() error {
			return c.client.Get(context.Background(), key, obj)
		}, opts.timeout).Should(gomega.Succeed())
	}
}
