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

package errors_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/errors"
)

var _ = Describe("Errors", func() {
	Describe("Re-exported methods", func() {
		It("exposes errors.New", func() {
			err := errors.New("test error")
			Expect(err).To(MatchError("test error"))
		})

		It("exposes errors.Errorf", func() {
			err := errors.Errorf("test %s", "error")
			Expect(err).To(MatchError("test error"))
		})

		It("exposes errors.Wrap", func() {
			err := errors.New("test error")
			err2 := errors.Wrap(err, "outer error")
			Expect(err2).To(MatchError("outer error: test error"))
		})

		It("exposes errors.Wrapf", func() {
			err := errors.New("test error")
			err2 := errors.Wrapf(err, "outer %s", "error")
			Expect(err2).To(MatchError("outer error: test error"))
		})

		It("exposes errors.Cause", func() {
			err := errors.New("test error")
			err2 := errors.Wrap(err, "outer error")
			Expect(errors.Cause(err2)).To(Equal(err))
		})
	})

	Describe("ShouldNotify", func() {
		It("returns true for a fmt.Errorf", func() {
			err := fmt.Errorf("test error")
			Expect(errors.ShouldNotify(err)).To(BeTrue())
		})

		It("returns true for an errors.New", func() {
			err := errors.Errorf("test error")
			Expect(errors.ShouldNotify(err)).To(BeTrue())
		})

		It("returns true for a errors.Errorf", func() {
			err := errors.Errorf("test error")
			Expect(errors.ShouldNotify(err)).To(BeTrue())
		})

		It("returns false for a top-level NoNotify", func() {
			err := errors.Errorf("test error")
			err2 := errors.NoNotify(err)
			Expect(errors.ShouldNotify(err2)).To(BeFalse())
			Expect(err2).To(MatchError("test error"))
			Expect(errors.Cause(err2)).To(Equal(err))
		})

		It("returns false for an intermediary NoNotify", func() {
			err := errors.Errorf("test error")
			err2 := errors.Wrap(errors.NoNotify(err), "outer error")
			Expect(errors.ShouldNotify(err2)).To(BeFalse())
			Expect(err2).To(MatchError("outer error: test error"))
			Expect(errors.Cause(err2)).To(Equal(err))
		})
	})
})
