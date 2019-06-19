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

package components

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	monitoringv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/monitoring/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
)

type logruleComponent struct {
}

func NewLogrule() *logruleComponent {
	return &logruleComponent{}
}

func (_ *logruleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *logruleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *logruleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	fmt.Println("in logrule")
	instance := ctx.Top.(*monitoringv1beta1.Monitor)
	rules, _ := yaml.Marshal(instance.Spec.LogAlertRules)
	fmt.Println("logrules", string(rules))
	// get folderid
	return components.Result{}, nil
}

func sumoRequest(url, method string, body []byte) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, baseURL+url, bytes.NewBuffer(body))
	req.SetBasicAuth("su5QkKDVw7s2AF", "i6KfitiTGzXlHXdtPDgBHrS3D0Dabogs6g6zeYOpsangmb9q1foXEbGZyijJa1dz")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	return respBody, nil
}

func getFolder(name string) (string, error) {
	resp, err := sumoRequest("/api/v2/content/folders/personal", "GET", nil)
	if err != nil {
		return "", err
	}
	js := &SumoFolderRsp{}
	_ = json.Unmarshal(resp, js)
	fmt.Println(js)
	return "", nil
}

const baseURL = "https://api.us2.sumologic.com/api/"

type SumoFolderRsp struct {
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy"`
	ModifiedAt  time.Time  `json:"modifiedAt"`
	ModifiedBy  string     `json:"modifiedBy"`
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	ItemType    string     `json:"itemType"`
	ParentID    string     `json:"parentId"`
	Permissions []string   `json:"permissions"`
	Description string     `json:"description"`
	Children    []Children `json:"children"`
}
type Children struct {
	CreatedAt   time.Time `json:"createdAt"`
	CreatedBy   string    `json:"createdBy"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	ModifiedBy  string    `json:"modifiedBy"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ItemType    string    `json:"itemType"`
	ParentID    string    `json:"parentId"`
	Permissions []string  `json:"permissions"`
}

type SearchQuery struct {
	Type             string          `json:"type"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	SearchQuery      string          `json:"searchQuery"`
	QueryParameters  []interface{}   `json:"queryParameters"`
	DefaultTimeRange string          `json:"defaultTimeRange"`
	ByReceiptTime    bool            `json:"byReceiptTime"`
	ViewNameOpt      string          `json:"viewNameOpt"`
	ViewStartTimeOpt int             `json:"viewStartTimeOpt"`
	Schedules        Schedules       `json:"schedules"`
	AutoParsingData  AutoParsingData `json:"autoParsingData"`
}
type ThresholdOption struct {
	Type     string `json:"type"`
	Operator string `json:"operator"`
	Count    int    `json:"count"`
}
type Notification struct {
	Type      string `json:"type"`
	WebhookID int    `json:"webhookId"`
	Payload   string `json:"payload"`
}
type Schedules struct {
	Type                 string          `json:"type"`
	CronSchedule         string          `json:"cronSchedule"`
	DisplayableTimeRange string          `json:"displayableTimeRange"`
	ParseableTimeRange   string          `json:"parseableTimeRange"`
	TimeZone             string          `json:"timeZone"`
	ThresholdOption      ThresholdOption `json:"thresholdOption"`
	Notification         Notification    `json:"notification"`
	ScheduleType         string          `json:"scheduleType"`
	CreatedBy            int             `json:"createdBy"`
	MuteErrorEmails      bool            `json:"muteErrorEmails"`
	QueryParameters      []string        `json:"queryParameters"`
}
type AutoParsingData struct {
	Mode string `json:"mode"`
}

var defaultsearch = &SearchQuery{
	Type:             "Search",
	Name:             "default name",
	Description:      "default Description",
	SearchQuery:      "*",
	DefaultTimeRange: "-5m",
	ByReceiptTime:    false,
	ViewNameOpt:      "",
	ViewStartTimeOpt: 0,
	Schedules: Schedules{
		Type:                 "SearchSchedule",
		CronSchedule:         "17 * * * * ? *",
		DisplayableTimeRange: "-5m",
		ParseableTimeRange:   "[{\"t\":\"relative\",\"d\":-300000}]",
		TimeZone:             "Etc/UTC",
		ThresholdOption: ThresholdOption{
			Type:     "GroupThreshold",
			Operator: "gt",
			Count:    0,
		},
		Notification: Notification{
			Type:      "WebhookSearchNotification",
			WebhookID: 39232,
			Payload:   "{\r\n \t\"attachments\": [\r\n \t\t{\r\n \t\t\t\"pretext\": \"Sumo Logic Alert: *{{SearchName}}*\",\r\n \t\t\t\"fields\": [\r\n \t\t\t\t{\r\n \t\t\t\t\t\"title\": \"Description\",\r\n \t\t\t\t\t\"value\": \"{{SearchDescription}}\"\r\n \t\t\t\t},\r\n \t\t\t\t{\r\n \t\t\t\t\t\"title\": \"Query\",\r\n \t\t\t\t\t\"value\": \"<{{SearchQueryUrl}}>\"\r\n \t\t\t\t},\r\n \t\t\t\t{\r\n \t\t\t\t\t\"title\": \"Time Range\",\r\n \t\t\t\t\t\"value\": \"{{TimeRange}}\"\r\n \t\t\t\t}\r\n \t\t\t],\r\n \t\t\t\"mrkdwn_in\": [\"text\", \"pretext\"],\r\n \t\t\t\"color\": \"#ff0000\"\r\n \t\t}\r\n \t]\r\n }",
		},
		ScheduleType:    "Real time",
		MuteErrorEmails: false,
	},
	AutoParsingData: AutoParsingData{
		Mode: "performance",
	},
}
