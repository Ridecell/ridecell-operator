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

package djangouser_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/controller/djangouser"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var testHelpers *test_helpers.TestHelpers

func TestTemplates(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "DjangoUser controller Suite @controller")
}

var _ = ginkgo.BeforeSuite(func() {
	testHelpers = test_helpers.Start(djangouser.Add, false)
})

var _ = ginkgo.AfterSuite(func() {
	testHelpers.Stop()
})
