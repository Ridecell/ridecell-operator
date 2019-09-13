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

package fake_pagerduty

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func RequestLogger(targetMux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		targetMux.ServeHTTP(w, r)

		requesterIP := r.RemoteAddr

		log.Printf(
			"%s\t\t%s\t\t%s\t\t%v",
			r.Method,
			r.RequestURI,
			requesterIP,
			time.Since(start),
		)
	})
}

func escalation_policies(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile(os.Getenv("PWD") + "/pkg/test_helpers/fake_pagerduty/ListEscalationPoliciesResponse.json")
	_, err := w.Write([]byte(file))
	if err != nil {
		log.Fatal(err)
	}

}

func services(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile(os.Getenv("PWD") + "/pkg/test_helpers/fake_pagerduty/ListServiceResponse.json")
	_, err := w.Write([]byte(file))
	if err != nil {
		log.Fatal(err)
	}

}

func Run() {
	log.SetOutput(os.Stdout)
	mux := http.NewServeMux()
	mux.HandleFunc("/escalation_policies", escalation_policies)
	mux.HandleFunc("/services", services)
	go func() {
		log.Println(http.ListenAndServe(":8082", RequestLogger(mux)))
	}()
}
