package taskservice

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"
)

type testConnector struct{ connection *testConnection }

func (c testConnector) Connect(context.Context) (driver.Conn, error) { return c.connection, nil }
func (c testConnector) Driver() driver.Driver                        { return testDriver{} }

type testDriver struct{}

func (testDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type testConnection struct {
	query string
	title string
}

func (c *testConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not implemented")
}
func (c *testConnection) Close() error { return nil }
func (c *testConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("begin is not implemented")
}
func (c *testConnection) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.query = query
	c.title = args[0].Value.(string)
	return &testRows{values: []driver.Value{int64(7), c.title}}, nil
}

type testRows struct {
	values []driver.Value
	done   bool
}

func (r *testRows) Columns() []string { return []string{"id", "title"} }
func (r *testRows) Close() error      { return nil }
func (r *testRows) Next(destination []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	copy(destination, r.values)
	return nil
}

func TestSQLStoreCreatesTaskWithParameterizedQuery(t *testing.T) {
	connection := &testConnection{}
	database := sql.OpenDB(testConnector{connection: connection})
	t.Cleanup(func() { _ = database.Close() })

	task, err := (&SQLStore{DB: database}).Create(t.Context(), "проверить запрос")
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != 7 || task.Title != "проверить запрос" {
		t.Fatalf("task = %+v", task)
	}
	if !strings.Contains(connection.query, "VALUES ($1)") || connection.title != task.Title {
		t.Fatalf("query=%q title=%q", connection.query, connection.title)
	}
}

func TestSQLStoreRequiresDatabase(t *testing.T) {
	_, err := (&SQLStore{}).Create(t.Context(), "задача")
	if err == nil {
		t.Fatal("expected configuration error")
	}
}
