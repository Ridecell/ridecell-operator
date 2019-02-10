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

package errors

import (
	"github.com/pkg/errors"
)

// Re-export all the important methods from pkg/errors.
var New = errors.New
var Errorf = errors.Errorf
var Wrap = errors.Wrap
var Wrapf = errors.Wrapf
var Cause = errors.Cause

// Because errors doesn't expose this.
type causer interface {
	Cause() error
}

// An error that should not get a notification.
func NoNotify(err error) error {
	return &noNotify{error: err}
}

type noNotify struct{ error }

func (n *noNotify) Cause() error {
	c, ok := n.error.(causer)
	if ok {
		return c.Cause()
	} else {
		return n.error
	}
}

func ShouldNotify(err error) bool {
	for err != nil {
		_, ok := err.(*noNotify)
		if ok {
			return false
		}
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return true
}
