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

package v1beta1

import (
	"github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
)

const (
	StatusReady     = "Ready"
	StatusCreating  = "Creating"
	StatusModifying = "Modifying"
	StatusError     = "Error"
	StatusUnknown   = "Unknown"
)

// Connection details for a Postgres database.
type PostgresConnection struct {
	Host              string            `json:"host"`
	Port              int               `json:"port,omitempty"`
	Username          string            `json:"username"`
	PasswordSecretRef helpers.SecretRef `json:"passwordSecretRef"`
	Database          string            `json:"database,omitempty"`
}

type RabbitmqConnection struct {
	Username     string            `json:"username"`
	Password     helpers.SecretRef `json:"passwordSecretRef"`
	Host         string            `json:"host"`
	InsecureSkip bool              `json:"insecureSkip,omitempty"` // skip server certificate trust, by default false
}
