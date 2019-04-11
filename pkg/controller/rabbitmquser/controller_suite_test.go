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

package rabbitmquser_test

import (
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/controller/rabbitmquser"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var testHelpers *test_helpers.TestHelpers

func TestController(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "RabbitMQ User controller Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	testHelpers = test_helpers.Start(rabbitmquser.Add, true)
})

var _ = ginkgo.AfterSuite(func() {
	// Wait to make sure any running reconciles have finished.
	time.Sleep(5 * time.Second)

	testHelpers.Stop()
})
