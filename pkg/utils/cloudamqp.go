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
	"github.com/Ridecell/ridecell-operator/pkg/errors"
	"net/http"
)

type Rule struct {
	Services    []string `json:"services"`
	IP          string   `json:"ip"`
	Description string   `json:"description"`
}

func PutCloudamqpFirewallRules(apiUrl string, apiKey string, data []Rule) error {

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)
	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth("", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return errors.Errorf("CLOUDAMQP firewall response code HTTP %d", resp.StatusCode)
	}
	return nil
}

func GetCloudamqpFirewallRules(apiUrl string, apiKey string) ([]Rule, error) {

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("CLOUDAMQP firewall response code HTTP %d", resp.StatusCode)
	}

	var rules []Rule
	err = json.NewDecoder(resp.Body).Decode(&rules)
	if err != nil {
		return nil, err
	}
	return rules, nil
}
