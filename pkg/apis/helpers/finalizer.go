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

package helpers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TODO: Replace this helper with a proper finalizer handler

// ContainsFinalizer checks if specified finalizer is attached to our object
func ContainsFinalizer(input string, obj runtime.Object) bool {
	for _, i := range getFinalizers(obj) {
		if i == input {
			return true
		}
	}
	return false
}

// AppendFinalizer returns new slice of finalizers with specified finalizer appended
func AppendFinalizer(input string, obj runtime.Object) []string {
	finalizers := getFinalizers(obj)
	finalizers = append(finalizers, input)
	return finalizers
}

// RemoveFinalizer returns new slice of finalizers with specified finalizer removed
func RemoveFinalizer(input string, obj runtime.Object) []string {
	var outputSlice []string
	for _, i := range getFinalizers(obj) {
		if i == input {
			continue
		}
		outputSlice = append(outputSlice, i)
	}
	return outputSlice
}

func getFinalizers(obj runtime.Object) []string {
	metaObject := obj.(metav1.Object)
	return metaObject.GetFinalizers()
}
