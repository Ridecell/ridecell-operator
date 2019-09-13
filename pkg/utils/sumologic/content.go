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
	"time"

	"github.com/pkg/errors"
)

type SavedSearchWithSchedule struct {
	Type            string          `json:"type"`
	Name            string          `json:"name"`
	Search          Search          `json:"search"`
	SearchSchedule  SearchSchedule  `json:"searchSchedule"`
	Description     string          `json:"description"`
	AutoParsingData AutoParsingData `json:"autoParsingData"`
}
type AutoParsingData struct {
	Mode string `json:"mode"`
}

type AutoCompleteValues struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type AutoComplete struct {
	AutoCompleteType   string               `json:"autoCompleteType"`
	AutoCompleteValues []AutoCompleteValues `json:"autoCompleteValues"`
	LookupFileName     string               `json:"lookupFileName"`
	LookupLabelColumn  string               `json:"lookupLabelColumn"`
	LookupValueColumn  string               `json:"lookupValueColumn"`
}
type QueryParameters struct {
	Name         string       `json:"name"`
	Label        string       `json:"label"`
	Description  string       `json:"description"`
	DataType     string       `json:"dataType"`
	Value        string       `json:"value"`
	AutoComplete AutoComplete `json:"autoComplete"`
}
type Search struct {
	QueryText        string            `json:"queryText"`
	DefaultTimeRange string            `json:"defaultTimeRange"`
	ByReceiptTime    bool              `json:"byReceiptTime"`
	ViewName         string            `json:"viewName"`
	ViewStartTime    time.Time         `json:"viewStartTime"`
	QueryParameters  []QueryParameters `json:"queryParameters"`
}
type From struct {
	RelativeTime string `json:"relativeTime"`
	Type         string `json:"type"`
}
type ParseableTimeRange struct {
	Type string      `json:"type"`
	From From        `json:"from"`
	To   interface{} `json:"to"`
}
type Threshold struct {
	ThresholdType string `json:"thresholdType"`
	Operator      string `json:"operator"`
	Count         int64  `json:"count"`
}
type Notification struct {
	TaskType  string `json:"taskType"`
	WebhookId string `json:"webhookId"`
	Payload   string `json:"payload"`
}
type Parameters struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type SearchSchedule struct {
	CronExpression       string             `json:"cronExpression"`
	DisplayableTimeRange string             `json:"displayableTimeRange"`
	ParseableTimeRange   ParseableTimeRange `json:"parseableTimeRange"`
	TimeZone             string             `json:"timeZone"`
	Threshold            Threshold          `json:"threshold"`
	Notification         Notification       `json:"notification"`
	ScheduleType         string             `json:"scheduleType"`
	MuteErrorEmails      bool               `json:"muteErrorEmails"`
	Parameters           []Parameters       `json:"parameters"`
}

type JobStatus struct {
	Status        string      `json:"status"`
	StatusMessage string      `json:"statusMessage"`
	Error         StatusError `json:"error"`
}
type StatusMeta struct {
	MinLength    int `json:"minLength"`
	ActualLength int `json:"actualLength"`
}
type StatusError struct {
	Code    string     `json:"code"`
	Message string     `json:"message"`
	Detail  string     `json:"detail"`
	Meta    StatusMeta `json:"meta"`
}

type Job struct {
	ID string `json:"id"`
}

type JobResult struct {
	Type        string        `json:"type"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Children    []interface{} `json:"children"`
}

// GetContent Get sumo content
func (c *Client) GetContent(id string) (*JobResult, error) {
	var jobresult JobResult
	// Create export Job
	req, err := c.newRequest("POST", "/api/v2/content/"+id+"/export", nil)
	if err != nil {
		return nil, err
	}
	var job Job
	_, err = c.do(req, &job)
	if err != nil {
		return &jobresult, err
	}

	// Check Job status
	var status JobStatus
	req, err = c.newRequest("GET", "/api/v2/content/"+id+"/export/"+job.ID+"/status", nil)
	if err != nil {
		return nil, err
	}
	for index := 0; status.Status != "Success"; index++ {
		_, err = c.do(req, &status)
		if err != nil {
			return &jobresult, errors.Wrapf(err, "Failed get status for job ID %s", job.ID)
		}
		if index > 5 {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if status.Status != "Success" {
		return &jobresult, errors.Errorf("Job status is %s %+v", status.Status, status)
	}
	// Get job result
	req, err = c.newRequest("GET", "/api/v2/content/"+id+"/export/"+job.ID+"/result", nil)
	if err != nil {
		return &jobresult, err
	}
	_, err = c.do(req, &jobresult)
	return &jobresult, err
}

func (c *Client) CreateSavedSearchWithSchedule(id string, search *SavedSearchWithSchedule, overwrite bool) (*JobStatus, error) {
	var status JobStatus
	// Create import search Job
	req, err := c.newRequest("POST", "/api/v2/content/folders/"+id+"/import", search)
	if err != nil {
		return nil, err
	}
	// Add quey parameters
	if overwrite {
		q := req.URL.Query()
		q.Add("overwrite", "true")
		req.URL.RawQuery = q.Encode()
	}
	var job Job
	_, err = c.do(req, &job)
	if err != nil {
		return &status, err
	}

	// Check import search Job status
	req, err = c.newRequest("GET", "/api/v2/content/folders/"+id+"/import/"+job.ID+"/status", nil)
	for index := 0; status.Status != "Success"; index++ {
		_, err = c.do(req, &status)
		if err != nil {
			return &status, errors.Wrapf(err, "Failed get status for job ID %s", job.ID)
		}
		if index > 5 {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if status.Status != "Success" {
		return &status, errors.Errorf("Job status is %s %+v", status.Status, status)
	}
	return &status, err
}

// DeleteContent delete content from sumo
func (c *Client) DeleteContent(id string) (*JobStatus, error) {
	var status JobStatus
	// Create import search Job
	req, err := c.newRequest("DELETE", "/api/v2/content/"+id+"/delete", nil)
	if err != nil {
		return nil, err
	}
	var job Job
	_, err = c.do(req, &job)
	if err != nil {
		return &status, err
	}
	// Check delete Job status
	req, err = c.newRequest("GET", "/api/v2/content/"+id+"/delete/"+job.ID+"/status", nil)
	for index := 0; status.Status != "Success"; index++ {
		_, err = c.do(req, &status)
		if err != nil {
			return &status, errors.Wrapf(err, "Failed get status for job ID %s", job.ID)
		}
		if index > 5 {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if status.Status != "Success" {
		return &status, errors.Errorf("Job status is %s %+v", status.Status, status)
	}
	return &status, err
}
