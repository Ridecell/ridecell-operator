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

package fake_sumologic

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

func connections(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile(os.Getenv("PWD") + "/pkg/test_helpers/fake_sumologic/connection.json")
	_, err := w.Write([]byte(file))
	if err != nil {
		log.Fatal(err)
	}

}

func folders(w http.ResponseWriter, r *http.Request) {
	file, _ := ioutil.ReadFile(os.Getenv("PWD") + "/pkg/test_helpers/fake_sumologic/folders.json")
	_, err := w.Write([]byte(file))
	if err != nil {
		log.Fatal(err)
	}

}

func import_folder(w http.ResponseWriter, r *http.Request) {
	resp := `{
		"id": "52E4451888457A51"
		}`
	_, err := w.Write([]byte(resp))
	if err != nil {
		log.Fatal(err)
	}

}

func job_status(w http.ResponseWriter, r *http.Request) {
	resp := `{
		"status": "Success",
		"statusMessage": null,
		"error": null
	}`
	_, err := w.Write([]byte(resp))
	if err != nil {
		log.Fatal(err)
	}

}
func Run() {
	log.SetOutput(os.Stdout)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/connections", connections)
	mux.HandleFunc("/api/v2/content/folders/000000000083D019", folders)
	mux.HandleFunc("/api/v2/content/folders/0000000000A16709/import", import_folder)
	mux.HandleFunc("/api/v2/content/folders/0000000000A16709/import/52E4451888457A51/status", job_status)
	go func() {
		log.Println(http.ListenAndServe(":8083", RequestLogger(mux)))
	}()
}
