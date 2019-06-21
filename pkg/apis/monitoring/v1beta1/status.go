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

package v1beta1

import (
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

func (amc *AlertManagerConfig) GetStatus() components.Status {
	return amc.Status
}

func (amc *AlertManagerConfig) SetStatus(status components.Status) {
	amc.Status = status.(AlertManagerConfigStatus)
}

func (amc *AlertManagerConfig) SetErrorStatus(errorMsg string) {
	amc.Status.Status = StatusError
	amc.Status.Message = errorMsg
}

func (mon *Monitor) GetStatus() components.Status {
	return mon.Status
}

func (mon *Monitor) SetStatus(status components.Status) {
	mon.Status = status.(MonitorStatus)
}

func (mon *Monitor) SetErrorStatus(errorMsg string) {
	mon.Status.Status = StatusError
	mon.Status.Message = errorMsg
}
