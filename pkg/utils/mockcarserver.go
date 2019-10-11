/*
Copyright 2018-2019 Ridecell, Inc.

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

package utils

import (
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Get the mock tenant
// GET request
// query param: name
// response code: 200 success (present), 404 (not found), 401 (invalid auth)
func GetMockTenant(tenantName string) (bool, error) {
	URI := os.Getenv("MOCKCARSERVER_URI")
	AUTH := os.Getenv("MOCKCARSERVER_AUTH")
	AUTH_CLIENT := "ridecell-operator"

	RETRY_COUNT := 3
	for RETRY_COUNT > 0 {
		client := http.Client{
			Timeout: 30 * time.Second,
		}
		request, err := http.NewRequest("GET", URI+"/common/tenant?name="+tenantName, nil)
		if err != nil {
			return false, errors.Wrapf(err, "Unable to create request.")
		}
		request.Header.Set("AUTH-KEY", AUTH)
		request.Header.Set("AUTH-CLIENT", AUTH_CLIENT)
		resp, err := client.Do(request)
		if err != nil {
			return false, errors.Wrapf(err, "Something bad happened while connecting to Mock car server.")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return true, nil
		} else if resp.StatusCode == 401 {
			return false, errors.New("Request not authorized.")
		} else if resp.StatusCode == 400 {
			return false, errors.New("Bad request to server")
		}
		// request interval
		time.Sleep(10 * time.Second)
		RETRY_COUNT -= 1
	}
	return false, errors.New("Unable to get mock car server tenant")
}

// Create the mock tenant
// POST request
// param: name, callbackUrl, tenantHardwareType, apiKey, secretKey, apiToken, pushApiKey, pushSecretKey, pushToken
// response code: 201 created, 400 (bad params), 401 (invalid auth)
func CreateOrUpdateMockTenant(postData map[string]string) (bool, error) {
	URI := os.Getenv("MOCKCARSERVER_URI")
	AUTH := os.Getenv("MOCKCARSERVER_AUTH")
	AUTH_CLIENT := "ridecell-operator"

	formData := url.Values{}
	for k, v := range postData {
		formData.Add(k, v)
	}

	RETRY_COUNT := 3
	for RETRY_COUNT > 0 {
		client := http.Client{
			Timeout: 30 * time.Second,
		}
		request, err := http.NewRequest("POST", URI+"/common/tenant", strings.NewReader(formData.Encode()))
		if err != nil {
			return false, errors.Wrapf(err, "Unable to create request.")
		}
		request.Header.Set("AUTH-KEY", AUTH)
		request.Header.Set("Content-type", "application/x-www-form-urlencoded")
		request.Header.Set("AUTH-CLIENT", AUTH_CLIENT)
		resp, err := client.Do(request)
		if err != nil {
			return false, errors.Wrapf(err, "Something bad happened while connecting to Mock car server.")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 201 || resp.StatusCode == 200 {
			return true, nil
		} else if resp.StatusCode == 401 {
			return false, errors.New("Request not authorized.")
		} else if resp.StatusCode == 400 {
			return false, errors.New("Bad request to server")
		}
		// request interval
		time.Sleep(10 * time.Second)
		RETRY_COUNT -= 1
	}
	return false, errors.New("Unable to create/update mock car server tenant")
}

// Delete the mock tenant
// DELETE request
// query param: name
// response code: 200 success, 400 (bad params), 401 (invalid auth)
func DeleteMockTenant(tenantName string) (bool, error) {
	URI := os.Getenv("MOCKCARSERVER_URI")
	AUTH := os.Getenv("MOCKCARSERVER_AUTH")
	AUTH_CLIENT := "ridecell-operator"

	RETRY_COUNT := 3
	for RETRY_COUNT > 0 {
		client := http.Client{
			Timeout: 30 * time.Second,
		}
		request, err := http.NewRequest("DELETE", URI+"/common/tenant?name="+tenantName, nil)
		if err != nil {
			return false, errors.Wrapf(err, "Unable to create request.")
		}
		request.Header.Set("AUTH-KEY", AUTH)
		request.Header.Set("Content-type", "application/x-www-form-urlencoded")
		request.Header.Set("AUTH-CLIENT", AUTH_CLIENT)
		resp, err := client.Do(request)
		if err != nil {
			return false, errors.Wrapf(err, "Something bad happened while connecting to Mock car server.")
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return true, nil
		} else if resp.StatusCode == 401 {
			return false, errors.New("Request not authorized.")
		} else if resp.StatusCode == 400 {
			return false, errors.New("Bad request to server")
		}
		// request interval
		time.Sleep(10 * time.Second)
		RETRY_COUNT -= 1
	}
	return false, errors.New("Unable to delete tenant on mock car server")
}
