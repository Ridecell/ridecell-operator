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

func (sb *S3Bucket) GetStatus() components.Status {
	return sb.Status
}

func (sb *S3Bucket) SetStatus(status components.Status) {
	sb.Status = status.(S3BucketStatus)
}

func (sb *S3Bucket) SetErrorStatus(errorMsg string) {
	sb.Status.Status = StatusError
	sb.Status.Message = errorMsg
}

func (iu *IAMUser) GetStatus() components.Status {
	return iu.Status
}

func (iu *IAMUser) SetStatus(status components.Status) {
	iu.Status = status.(IAMUserStatus)
}

func (iu *IAMUser) SetErrorStatus(errorMsg string) {
	iu.Status.Status = StatusError
	iu.Status.Message = errorMsg
}

func (es *ElasticSearch) GetStatus() components.Status {
	return es.Status
}

func (es *ElasticSearch) SetStatus(status components.Status) {
	es.Status = status.(ElasticSearchStatus)
}

func (es *ElasticSearch) SetErrorStatus(errorMsg string) {
	es.Status.Status = StatusError
	es.Status.Message = errorMsg
}
