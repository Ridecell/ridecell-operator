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

package fake_mockcarserver

import (
	"log"
	"net/http"
	"os"
	"time"
)

func RequestLogger(targetMux http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		targetMux.ServeHTTP(w, r)

		// log request by who(IP address)
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

func tenant(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// create tenant
		w.WriteHeader(201)
	} else if r.Method == "GET" {
		// update tenant
		w.WriteHeader(200)
	} else if r.Method == "DELETE" {
		// delete tenant
		w.WriteHeader(200)
	}
	resp := `response`
	_, err := w.Write([]byte(resp))
	if err != nil {
		log.Fatal(err)
	}
}
func Run() {
	log.SetOutput(os.Stdout)
	mux := http.NewServeMux()
	mux.HandleFunc("/common/tenant", tenant)
	go func() {
		log.Println(http.ListenAndServe(":9090", RequestLogger(mux)))
	}()
}
