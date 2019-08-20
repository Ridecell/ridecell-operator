// +build !release

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

package ridecellingress

import (
	"net/http"
	"path"
	"runtime"
)

//go:generate bash ../../../hack/assets_generate.sh controller/ridecellingress ridecellingress
var Templates http.FileSystem

func init() {
	_, line, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to find caller line")
	}
	Templates = http.Dir(path.Dir(line) + "/templates")
}
