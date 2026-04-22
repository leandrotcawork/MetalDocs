package authz

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

type authzTestState struct {
	granted         bool
	actorID         string
	assertedCaps    string
	requireQueries  int
	executedQueries []string
}

type authzTestDriver struct {
	state *authzTestState
}

func (d *authzTestDriver) Open(_ string) (driver.Conn, error) {
	return &authzTestConn{state: d.state}, nil
}

type authzTestConn struct {
	state *authzTestState
}

func (c *authzTestConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("Prepare not used")
}
func (c *authzTestConn) Close() error              { return nil }
func (c *authzTestConn) Begin() (driver.Tx, error) { return authzTestTx{}, nil }
func (c *authzTestConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return authzTestTx{}, nil
}

func (c *authzTestConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.executedQueries = append(c.state.executedQueries, query)

	if strings.Contains(query, "set_config('metaldocs.asserted_caps'") {
		if len(args) != 1 {
			return nil, fmt.Errorf("set_config asserted caps: got %d args, want 1", len(args))
		}
		val, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("set_config asserted caps arg type %T", args[0].Value)
		}
		c.state.assertedCaps = val
	}

	return driver.RowsAffected(1), nil
}

func (c *authzTestConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "SELECT EXISTS"):
		c.state.requireQueries++
		return &authzTestRows{
			columns: []string{"exists"},
			rows:    [][]driver.Value{{c.state.granted}},
		}, nil
	case strings.Contains(query, "current_setting('metaldocs.asserted_caps', true)"):
		return &authzTestRows{
			columns: []string{"current_setting"},
			rows:    [][]driver.Value{{c.state.assertedCaps}},
		}, nil
	case strings.Contains(query, "current_setting('metaldocs.actor_id', false)"):
		return &authzTestRows{
			columns: []string{"current_setting"},
			rows:    [][]driver.Value{{c.state.actorID}},
		}, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

type authzTestTx struct{}

func (authzTestTx) Commit() error   { return nil }
func (authzTestTx) Rollback() error { return nil }

type authzTestRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *authzTestRows) Columns() []string { return r.columns }
func (r *authzTestRows) Close() error      { return nil }
func (r *authzTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func openAuthzTestDB(t *testing.T, state *authzTestState) (*sql.DB, *sql.Tx) {
	t.Helper()

	name := fmt.Sprintf("authz_test_%p", state)
	sql.Register(name, &authzTestDriver{state: state})

	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return db, tx
}

func TestRequire_CapGranted(t *testing.T) {
	state := &authzTestState{
		granted:      true,
		actorID:      "actor-1",
		assertedCaps: `[]`,
	}
	_, tx := openAuthzTestDB(t, state)

	err := Require(WithCapCache(context.Background()), tx, "doc.submit", "AREA1")
	if err != nil {
		t.Fatalf("Require returned error: %v", err)
	}

	var got []map[string]string
	if err := json.Unmarshal([]byte(state.assertedCaps), &got); err != nil {
		t.Fatalf("Unmarshal asserted caps: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("asserted caps len = %d, want 1", len(got))
	}
	if got[0]["cap"] != "doc.submit" || got[0]["area"] != "AREA1" {
		t.Fatalf("asserted cap = %#v, want doc.submit/AREA1", got[0])
	}
}

func TestRequire_CapDenied(t *testing.T) {
	state := &authzTestState{
		granted: false,
		actorID: "actor-2",
	}
	_, tx := openAuthzTestDB(t, state)

	err := Require(WithCapCache(context.Background()), tx, "doc.publish", "AREA2")
	if err == nil {
		t.Fatal("Require returned nil, want ErrCapabilityDenied")
	}

	var denied ErrCapabilityDenied
	if !errors.As(err, &denied) {
		t.Fatalf("Require error = %T %v, want ErrCapabilityDenied", err, err)
	}
	if denied.Capability != "doc.publish" || denied.AreaCode != "AREA2" || denied.ActorID != "actor-2" {
		t.Fatalf("denied error = %#v", denied)
	}
}

func TestRequire_CacheHit(t *testing.T) {
	state := &authzTestState{
		granted:      true,
		actorID:      "actor-3",
		assertedCaps: `[]`,
	}
	_, tx := openAuthzTestDB(t, state)
	ctx := WithCapCache(context.Background())

	if err := Require(ctx, tx, "doc.signoff", "AREA3"); err != nil {
		t.Fatalf("first Require: %v", err)
	}
	if err := Require(ctx, tx, "doc.signoff", "AREA3"); err != nil {
		t.Fatalf("second Require: %v", err)
	}
	if state.requireQueries != 1 {
		t.Fatalf("require query count = %d, want 1", state.requireQueries)
	}
}

func TestBypassSystem(t *testing.T) {
	state := &authzTestState{}
	_, tx := openAuthzTestDB(t, state)

	if err := BypassSystem(context.Background(), tx); err != nil {
		t.Fatalf("BypassSystem returned error: %v", err)
	}

	found := false
	for _, query := range state.executedQueries {
		if query == "SELECT set_config('metaldocs.bypass_authz', 'scheduler', true)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("executed queries = %#v, want bypass set_config", state.executedQueries)
	}
}
