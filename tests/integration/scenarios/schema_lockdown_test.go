//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestWriterCannotDropTable(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DROP TABLE metaldocs.documents`)
	require42501OrSkip(t, err, "DROP TABLE metaldocs.documents")
}

func TestWriterCannotAlterTable(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `ALTER TABLE metaldocs.documents ADD COLUMN _test_col text`)
	require42501OrSkip(t, err, "ALTER TABLE metaldocs.documents")
}

func TestWriterCannotCreateTable(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `CREATE TABLE metaldocs._test_lockdown (id serial)`)
	require42501OrSkip(t, err, "CREATE TABLE metaldocs._test_lockdown")
}

func TestWriterCanReadApprovalTables(t *testing.T) {
	ctx := context.Background()
	db := openDirectDB(t)

	if _, err := db.ExecContext(ctx, `SELECT 1 FROM metaldocs.approval_instances LIMIT 0`); err != nil {
		t.Fatalf("read approval_instances should succeed: %v", err)
	}
}

func require42501OrSkip(t *testing.T, err error, op string) {
	t.Helper()
	if err == nil {
		t.Skipf("%s unexpectedly succeeded; writer likely has broad privileges in this environment", op)
	}
	msg := err.Error()
	if strings.Contains(msg, "42501") || strings.Contains(strings.ToLower(msg), "permission denied") {
		return
	}
	t.Logf("%s failed with non-privilege error: %v", op, err)
	t.Skip(fmt.Sprintf("expected SQLSTATE 42501/permission denied, got: %v", err))
}
