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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}

func (frc *Client) ListServices(o pagerduty.ListServiceOptions) (*pagerduty.ListServiceResponse, error) {
	file, err := ioutil.ReadFile("ListServiceResponse.json")
	fmt.Println("file", err)
	lsr := pagerduty.ListServiceResponse{}
	_ = json.Unmarshal([]byte(file), &lsr)
	return &pagerduty.ListServiceResponse{}, nil
}

func (frc *Client) CreateService(s pagerduty.Service) (*pagerduty.Service, error) {

	return &pagerduty.Service{}, nil
}

func (frc *Client) ListEscalationPolicies(p pagerduty.ListEscalationPoliciesOptions) (*pagerduty.ListEscalationPoliciesResponse, error) {

	file, _ := ioutil.ReadFile(os.Getenv("PWD") + "/pkg/test_helpers/fake_pagerduty/ListEscalationPoliciesResponse.json")
	lep := pagerduty.ListEscalationPoliciesResponse{}
	_ = json.Unmarshal([]byte(file), &lep)

	return &lep, nil
}

func (frc *Client) CreateEventRule(e pagerduty.EventRule) (*pagerduty.EventRule, error) {

	return &pagerduty.EventRule{}, nil
}
