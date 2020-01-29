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
	"regexp"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	summoncomponents "github.com/Ridecell/ridecell-operator/pkg/controller/summon/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

// Mocktags represents the state of gcr util's Cache Tag
var MockTags []string

// Fetches the latest tag from mocked cache tag, skiping internal cache time checks
// (i.e. gcr util's CacheExpiry and LastCacheUpdate).
func MockGetLatestImageOfBranch(bRegex string) (string, error) {
	var latestImage string
	latestBuild := 0

	for _, image := range MockTags {
		match, err := regexp.Match(regexp.QuoteMeta(bRegex)+"$", []byte(image))
		if err != nil {
			return "", errors.Wrapf(err, "regexp.Match(%s, []byte(%s)) in GetLatestImageOfBranch()", bRegex, image)
		}

		if match {
			// Expects docker image to follow format <circleci buildnum>-<git hash>-<branchname>
			buildNumStr := strings.Split(image, "-")[0]
			buildNum, err := strconv.Atoi(buildNumStr)
			if err != nil {
				// skip glog since we're mocking function.
				continue
			}
			// Check for largest buildNumStr instead of running a sort, since we're doing O(n) anyway.
			if buildNum > latestBuild {
				latestBuild = buildNum
				latestImage = image
			}
		}
	}
	return latestImage, nil
}

var _ = Describe("SummonPlatform AutoDeploy Component", func() {
	comp := summoncomponents.NewAutoDeploy()

	BeforeEach(func() {
		// Version and AutoDeploy should be exclusive.
		instance.Spec.Version = ""
		// Start each test case off with some test tags and reset cache timestamp to zero.
		MockTags = []string{"1-abc1234-test-branch", "2-def5678-test-branch", "1-abc1234-other-branch"}
		comp.InjectMockTagFetcher(MockGetLatestImageOfBranch)
	})

	Describe("isReconcilable", func() {
		It("returns false if autoDeploy is not set", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})

		It("returns true if autoDeploy is set", func() {
			instance.Spec.AutoDeploy = "test-branch"
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})

		It("returns false if Spec.Version is also set", func() {
			instance.Spec.AutoDeploy = "test-branch"
			instance.Spec.Version = "1.2.3"
			Expect(comp.IsReconcilable(ctx)).To(BeFalse())
		})
	})

	It("sets the image version to the latest tag seen in tag cache for the given branch", func() {
		instance.Spec.AutoDeploy = "test-branch"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Version).To(Equal("2-def5678-test-branch"))
		instance.Spec.AutoDeploy = "other-branch"
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Version).To(Equal("1-abc1234-other-branch"))
	})

	// Tag cache expires, so it gets updated.
	It("uses the latest image from the updated tag cache", func() {
		instance.Spec.AutoDeploy = "test-branch"
		// Mock updated tag cache values
		MockTags = []string{"1-abc1234-test-branch", "2-def5678-test-branch", "3-ghi9101112-test-branch", "1-abc1234-other-branch", "2-def5678-other-branch"}
		Expect(comp).To(ReconcileContext(ctx))
		Expect(instance.Spec.Version).To(Equal("3-ghi9101112-test-branch"))
	})

	It("leaves Spec.Version alone if no matching image found", func() {
		instance.Spec.AutoDeploy = "nonexistent-branch"
		_, err := comp.Reconcile(ctx)
		Expect(err).To(MatchError("autodeploy: no matching branch image for nonexistent-branch"))
		Expect(instance.Spec.Version).To(Equal(""))
	})
})
