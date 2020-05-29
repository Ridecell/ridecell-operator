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

type CloudamqpFirewallRule struct {
	Services    []string `json:"services"`
	IP          string   `json:"ip"`
	Description string   `json:"description"`
}

var rules []CloudamqpFirewallRule
var IPList []string

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

func firewall(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		responseBytes, err := json.Marshal(rules)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		_, err = w.Write(responseBytes)
		if err != nil {
			log.Fatal(err)
		}
	} else if r.Method == "POST" {
		err := json.NewDecoder(r.Body).Decode(&rules)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		IPList = nil
		for _, rule := range rules {
			IPList = append(IPList, rule.IP)
		}
		log.Printf("IPList:\t\t%s", IPList)
		w.WriteHeader(201)
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
