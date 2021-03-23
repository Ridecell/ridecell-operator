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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"net/http"
)

func CallGithubActionsWebhook(apiUrl string, apiKey string, data map[string]interface{}) error {

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println("JsonPostData: ", string(payloadBytes))
	body := bytes.NewReader(payloadBytes)
	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// req.Header.Set("Accept", "application/json")
	req.Header.Set("x-attribution-login", "ridecell-operator")
	req.Header.Set("x-attribution-actor-id", "ridecell-operator")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return errors.Errorf("GithubActions Regression webhook response code HTTP %d", resp.StatusCode)
	}
	return nil
}
