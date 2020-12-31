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
	"time"

	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	rdssnapshotcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/rdssnapshot/components"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("rds Defaults Component", func() {
	It("does nothing on a filled out object", func() {
		comp := rdssnapshotcomponents.NewDefaults()
		instance.Spec.SnapshotID = "test-name"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.SnapshotID).To(Equal("test-name"))

	})

	It("sets defaults", func() {
		comp := rdssnapshotcomponents.NewDefaults()
		timeLocationUTC, err := time.LoadLocation("UTC")
		Expect(err).ToNot(HaveOccurred())
		// Set creationTimestamp to a predictable value
		currentTime := metav1.Date(2000, time.January, 1, 0, 0, 0, 0, timeLocationUTC)
		instance.ObjectMeta.SetCreationTimestamp(currentTime)
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.SnapshotID).To(Equal("test-2000-01-01-00-00-00"))
	})test-qa__master-3.1

	It("sanitizes snapshot id", func() {
		comp := rdssnapshotcomponents.NewDefaults()
		instance.Spec.SnapshotID = "test-qa__master-3.1"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.SnapshotID).To(Equal("test-qa-master-3-1"))
	})

})
