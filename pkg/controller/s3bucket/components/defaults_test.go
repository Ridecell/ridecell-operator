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

package components_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	s3bucketcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/s3bucket/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("s3bucket Defaults Component", func() {
	It("does nothing on a filled out object", func() {
		comp := s3bucketcomponents.NewDefaults()
		instance.Spec.BucketName = "test"
		instance.Spec.Region = "notaregion"

		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.BucketName).To(Equal("test"))
		Expect(instance.Spec.Region).To(Equal("notaregion"))
	})

	It("sets defaults", func() {
		comp := s3bucketcomponents.NewDefaults()
		Expect(comp).To(ReconcileContext(ctx))

		Expect(instance.Spec.BucketName).To(Equal("test-bucket"))
		Expect(instance.Spec.Region).To(Equal("us-west-2"))
	})

})
