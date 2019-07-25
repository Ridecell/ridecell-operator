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

package v1beta1_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
)

var testHelpers *test_helpers.TestHelpers

func TestObject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RidecelIngress types")

}

var _ = BeforeSuite(func() {
	testHelpers = test_helpers.Start(nil, false)
})

var _ = AfterSuite(func() {
	testHelpers.Stop()
})
