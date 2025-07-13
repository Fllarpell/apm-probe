package apmsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"time"
)

// ---------------- Driver registration ----------------

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.Driver)
)

// Register wraps the provided driver with tracing logic and registers it in
// database/sql under the given name. Typical usage:
//
//   import "github.com/lib/pq"
//   apmsql.Register("postgres-apm", &pq.Driver{})
//   db, _ := sql.Open("postgres-apm", dsn)
//
// Panics if the driver is nil or the name is already taken.
func Register(name string, d driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()

	if d == nil {
		panic("apmsql: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("apmsql: Register called twice for driver " + name)
	}

	drivers[name] = d
	sql.Register(name, &apmDriver{realDriver: d})
}

// ---------------- Driver wrappers ----------------

type apmDriver struct{ realDriver driver.Driver }

func (d *apmDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.realDriver.Open(name)
	if err != nil {
		return nil, err
	}
	return &apmConn{realConn: conn}, nil
}

type apmConn struct{ realConn driver.Conn }

func (c *apmConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.realConn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &apmStmt{realStmt: stmt, query: query}, nil
}
func (c *apmConn) Close() error                         { return c.realConn.Close() }
func (c *apmConn) Begin() (driver.Tx, error)            { return c.realConn.Begin() }

// Context-aware exec/query
func (c *apmConn) QueryContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Rows, error) {
	if qx, ok := c.realConn.(driver.QueryerContext); ok {
		start := time.Now()
		rows, err := qx.QueryContext(ctx, q, a)
		recordQuery(ctx, q, time.Since(start))
		return rows, err
	}
	return nil, driver.ErrSkip
}
func (c *apmConn) ExecContext(ctx context.Context, q string, a []driver.NamedValue) (driver.Result, error) {
	if ex, ok := c.realConn.(driver.ExecerContext); ok {
		start := time.Now()
		res, err := ex.ExecContext(ctx, q, a)
		recordQuery(ctx, q, time.Since(start))
		return res, err
	}
	return nil, driver.ErrSkip
}

type apmStmt struct {
	realStmt driver.Stmt
	query    string
}

func (s *apmStmt) Close() error               { return s.realStmt.Close() }
func (s *apmStmt) NumInput() int              { return s.realStmt.NumInput() }
func (s *apmStmt) Exec(args []driver.Value) (driver.Result, error) { return s.realStmt.Exec(args) }
func (s *apmStmt) Query(args []driver.Value) (driver.Rows, error) { return s.realStmt.Query(args) }

func (s *apmStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if ex, ok := s.realStmt.(driver.StmtExecContext); ok {
		start := time.Now()
		res, err := ex.ExecContext(ctx, args)
		recordQuery(ctx, s.query, time.Since(start))
		return res, err
	}
	values := namedValueToValue(args)
	start := time.Now()
	res, err := s.realStmt.Exec(values)
	recordQuery(ctx, s.query, time.Since(start))
	return res, err
}

func (s *apmStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if qx, ok := s.realStmt.(driver.StmtQueryContext); ok {
		start := time.Now()
		rows, err := qx.QueryContext(ctx, args)
		recordQuery(ctx, s.query, time.Since(start))
		return rows, err
	}
	values := namedValueToValue(args)
	start := time.Now()
	rows, err := s.realStmt.Query(values)
	recordQuery(ctx, s.query, time.Since(start))
	return rows, err
}

func namedValueToValue(named []driver.NamedValue) []driver.Value {
	vs := make([]driver.Value, len(named))
	for i, nv := range named {
		vs[i] = nv.Value
	}
	return vs
}
