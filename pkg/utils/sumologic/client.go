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

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/pkg/errors"
)

var newDefaultHTTPClient http.Client = http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	},
}

// Serror standard error response form sumoapi
type Serror struct {
	ID     string          `json:"id"`
	Errors []SerrorDetails `json:"errors"`
}

// SerrorDetails detailed error response
type SerrorDetails struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail"`
	Meta    map[string]string `json:"meta"`
}

// StdResp
type StdResp struct {
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy"`
	ModifiedAt time.Time `json:"modifiedAt"`
	ModifiedBy string    `json:"modifiedBy"`
	ID         string    `json:"id"`
}

type Client struct {
	BaseURL    string
	Auth       string
	httpClient *http.Client
}

func NewClient(baseurl, id, key string) (*Client, error) {
	return &Client{
		BaseURL:    baseurl,
		Auth:       base64.StdEncoding.EncodeToString([]byte(id + ":" + key)),
		httpClient: &newDefaultHTTPClient,
	}, nil
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, c.BaseURL+path, buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+c.Auth)
	return req, nil
}
func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		err = json.NewDecoder(resp.Body).Decode(v)
	} else {
		var e Serror
		err := json.NewDecoder(resp.Body).Decode(&e)
		if err != nil {
			return resp, errors.Wrapf(err, "Unknown error")
		}
		return resp, errors.Errorf("Request failed with stats %d with %+v", resp.StatusCode, e)
	}
	return resp, err
}
