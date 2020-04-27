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

func (sa *GCPServiceAccount) GetStatus() components.Status {
	return sa.Status
}

func (sa *GCPServiceAccount) SetStatus(status components.Status) {
	sa.Status = status.(GCPServiceAccountStatus)
}

func (sa *GCPServiceAccount) SetErrorStatus(errorMsg string) {
	sa.Status.Status = StatusError
	sa.Status.Message = errorMsg
}

func (gp *GCPProject) GetStatus() components.Status {
	return gp.Status
}

func (gp *GCPProject) SetStatus(status components.Status) {
	gp.Status = status.(GCPProjectStatus)
}

func (gp *GCPProject) SetErrorStatus(errorMsg string) {
	gp.Status.Status = StatusError
	gp.Status.Message = errorMsg
}
