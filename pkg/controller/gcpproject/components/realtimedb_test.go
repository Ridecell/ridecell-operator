/*
Copyright 2020 Ridecell, Inc.

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
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	gppcomponents "github.com/Ridecell/ridecell-operator/pkg/controller/gcpproject/components"
	. "github.com/Ridecell/ridecell-operator/pkg/test_helpers/matchers"
)

var _ = Describe("gcpproject realtimedb Component", func() {
	comp := gppcomponents.NewRealtimeDB()
	var httpMock *httptest.Server
	var getCount int
	var putCount int
	BeforeEach(func() {
		comp = gppcomponents.NewRealtimeDB()

		getCount = 0
		putCount = 0
		httpMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				_, err := w.Write([]byte("{}"))
				if err != nil {
					panic(err)
				}
				getCount++
			}

			if r.Method == "PUT" {
				putCount++
			}
		}))
		comp.InjectHTTPClient(httpMock.Client(), httpMock.URL)
		instance.Spec.RealtimeDatabaseRules = `{}`
	})

	AfterEach(func() {
		httpMock.Close()
	})

	Describe("IsReconcilable", func() {
		It("returns true", func() {
			Expect(comp.IsReconcilable(ctx)).To(BeTrue())
		})
	})

	It("does nothing if the flag is not set", func() {
		Expect(comp).To(ReconcileContext(ctx))
		Expect(getCount).To(Equal(0))
		Expect(putCount).To(Equal(0))
	})

	Describe("with flag set", func() {
		BeforeEach(func() {
			trueBool := true
			instance.Spec.EnableFirebase = &trueBool
			instance.Spec.EnableRealtimeDatabase = &trueBool
		})

		It("adds rules to a new db", func() {
			instance.Spec.RealtimeDatabaseRules = `{"newRules": "test"}`
			Expect(comp).To(ReconcileContext(ctx))
			Expect(getCount).To(Equal(1))
			Expect(putCount).To(Equal(1))
		})

		It("does nothing to an existing database with matching rules", func() {
			instance.Spec.RealtimeDatabaseRules = `{"test": "dothetest"}`
			httpMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					_, err := w.Write([]byte(`{"test": "dothetest"}`))
					if err != nil {
						panic(err)
					}
					getCount++
				}

				if r.Method == "PUT" {
					putCount++
				}
			}))
			comp.InjectHTTPClient(httpMock.Client(), httpMock.URL)
			Expect(comp).To(ReconcileContext(ctx))
			Expect(getCount).To(Equal(1))
			Expect(putCount).To(Equal(0))
		})
	})
})
