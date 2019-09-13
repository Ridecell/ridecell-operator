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

package fake_sql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"regexp"
	"strings"
)

// This implements a hilariously minimal no-op SQL driver for Go. It entirely
// ignores all input and returns nothing but null values anywhere that matters.
// This is for use only in controller functional tests where sqlmock is a bad fit
// because queries run an unknown number of times and in an unpredictable order.
// Do not use it in component tests, use sqlmock instead so you can actually test
// your component.

type fakeDriver struct{}
type fakeConn struct{}
type fakeStmt struct{ query string }
type fakeResult struct{}
type fakeRows struct {
	columns int
	eof     bool
}

var numInputs *regexp.Regexp
var cols *regexp.Regexp

func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	return &fakeConn{}, nil
}

func init() {
	numInputs = regexp.MustCompile(`\$\d+`)
	cols = regexp.MustCompile(`(?i:select (.*) from)`)
	sql.Register("fake_sql", &fakeDriver{})
}

func Open() *sql.DB {
	conn, _ := sql.Open("fake_sql", "")
	return conn
}

func (_ *fakeConn) Begin() (driver.Tx, error) {
	return nil, nil
}

func (_ *fakeConn) Close() error {
	return nil
}

func (_ *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{query: query}, nil
}

func (_ *fakeStmt) Close() error {
	return nil
}

func (s *fakeStmt) NumInput() int {
	return len(numInputs.FindAllString(s.query, -1))
}

func (_ *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &fakeResult{}, nil
}

func (_ *fakeStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return &fakeResult{}, nil
}

func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.fakeRows(), nil
}

func (s *fakeStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.fakeRows(), nil
}

func (s *fakeStmt) fakeRows() *fakeRows {
	// Is this the world's worst SQL parser? Possibly.
	colsSection := cols.FindString(s.query)
	commaCount := strings.Count(colsSection, ",")
	return &fakeRows{columns: commaCount + 1}
}

func (_ *fakeResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (_ *fakeResult) RowsAffected() (int64, error) {
	return 0, nil
}

func (r *fakeRows) Columns() []string {
	return make([]string, r.columns)
}

func (_ *fakeRows) Close() error {
	return nil
}

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.eof {
		return io.EOF
	} else {
		// One fake row
		r.eof = true
		for i := range dest {
			dest[i] = 0
		}
		return nil
	}
}

func (_ *fakeRows) ColumnTypeNullable(_ int) (bool, bool) {
	return true, true
}
