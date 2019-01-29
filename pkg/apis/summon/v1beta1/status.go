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
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

func (s *SummonPlatform) GetStatus() components.Status {
	return s.Status
}

func (s *SummonPlatform) SetStatus(status components.Status) {
	s.Status = status.(SummonPlatformStatus)
}

func (s *SummonPlatform) SetErrorStatus(errorMsg string) {
	s.Status.Status = StatusError
	s.Status.Message = errorMsg
}
