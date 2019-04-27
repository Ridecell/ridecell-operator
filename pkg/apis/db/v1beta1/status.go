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

func (pe *PostgresExtension) GetStatus() components.Status {
	return pe.Status
}

func (pe *PostgresExtension) SetStatus(status components.Status) {
	pe.Status = status.(PostgresExtensionStatus)
}

func (pe *PostgresExtension) SetErrorStatus(errorMsg string) {
	pe.Status.Status = StatusError
	pe.Status.Message = errorMsg
}

func (po *PostgresOperatorDatabase) GetStatus() components.Status {
	return po.Status
}

func (po *PostgresOperatorDatabase) SetStatus(status components.Status) {
	po.Status = status.(PostgresOperatorDatabaseStatus)
}

func (po *PostgresOperatorDatabase) SetErrorStatus(errorMsg string) {
	po.Status.Status = StatusError
	po.Status.Message = errorMsg
}

func (rds *RDSInstance) GetStatus() components.Status {
	return rds.Status
}

func (rds *RDSInstance) SetStatus(status components.Status) {
	rds.Status = status.(RDSInstanceStatus)
}

func (rds *RDSInstance) SetErrorStatus(errorMsg string) {
	rds.Status.Status = StatusError
	rds.Status.Message = errorMsg
}

func (pgu *PostgresUser) GetStatus() components.Status {
	return pgu.Status
}

func (pgu *PostgresUser) SetStatus(status components.Status) {
	pgu.Status = status.(PostgresUserStatus)
}

func (pgu *PostgresUser) SetErrorStatus(errorMsg string) {
	pgu.Status.Status = StatusError
	pgu.Status.Message = errorMsg
}

func (pgu *PostgresDatabase) GetStatus() components.Status {
	return pgu.Status
}

func (pgu *PostgresDatabase) SetStatus(status components.Status) {
	pgu.Status = status.(PostgresDatabaseStatus)
}

func (pgu *PostgresDatabase) SetErrorStatus(errorMsg string) {
	pgu.Status.Status = StatusError
	pgu.Status.Message = errorMsg
}

func (pgu *DbConfig) GetStatus() components.Status {
	return pgu.Status
}

func (pgu *DbConfig) SetStatus(status components.Status) {
	pgu.Status = status.(DbConfigStatus)
}

func (pgu *DbConfig) SetErrorStatus(errorMsg string) {
	pgu.Status.Status = StatusError
	pgu.Status.Message = errorMsg
}
