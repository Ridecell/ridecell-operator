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

package sumologic

type ListConnectionResp struct {
	Data []WebHookConnection `json:"data"`
	Next bool                `json:"next"`
}
type WebHookHeaders struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type WebHookConnection struct {
	StdResp
	Type           string           `json:"type"`
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	URL            string           `json:"url"`
	Headers        []WebHookHeaders `json:"headers"`
	CustomHeaders  []interface{}    `json:"customHeaders"`
	DefaultPayload string           `json:"defaultPayload"`
	WebhookType    string           `json:"webhookType"`
}

// ListConnections List all connections
func (c *Client) ListConnections() (*ListConnectionResp, error) {
	req, err := c.newRequest("GET", "/api/v1/connections", nil)
	var connectiosn ListConnectionResp
	if err != nil {
		return &connectiosn, err
	}

	_, err = c.do(req, &connectiosn)

	return &connectiosn, err
}

// GetConnection get connection
func (c *Client) GetConnection(id string) (*WebHookConnection, error) {
	req, err := c.newRequest("GET", "/api/v1/connections/"+id, nil)
	var connectiosn WebHookConnection
	if err != nil {
		return &connectiosn, err
	}
	_, err = c.do(req, &connectiosn)
	return &connectiosn, err
}

// CreateConnection create webhook connection
func (c *Client) CreateConnection(webhook WebHookConnection) (*WebHookConnection, error) {
	req, err := c.newRequest("POST", "/api/v1/connections", webhook)
	var connectiosn WebHookConnection
	if err != nil {
		return &connectiosn, err
	}
	_, err = c.do(req, &connectiosn)
	return &connectiosn, err
}
