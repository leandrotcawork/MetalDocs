//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"
)

// TestTriggerBypassBlocked verifies that the writer role cannot bypass
// the legal-transition trigger via session_replication_role.
func TestTriggerBypassBlocked(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `SET LOCAL session_replication_role = 'replica'`)
	if err == nil {
		t.Log("SET session_replication_role succeeded; checking illegal transition still blocked")

		tenantID := testdb.DeterministicID(t, "tenant")
		docID := testdb.DeterministicID(t, "doc")
		userID := testdb.DeterministicID(t, "user")
		fixtures.SeedUser(t, ctx, db, schema, userID, "Trigger User")
		fixtures.SeedDocument(t, ctx, db, schema, docID, tenantID, userID)

		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s
			   SET status = 'published'
			 WHERE id = $1::uuid AND tenant_id = $2::uuid`,
			testdb.Qualified(schema, "documents")),
			docID, tenantID,
		); err == nil {
			var status string
			if err := tx.QueryRowContext(ctx, fmt.Sprintf(
				`SELECT status FROM %s WHERE id = $1::uuid`,
				testdb.Qualified(schema, "documents"),
			), docID).Scan(&status); err != nil {
				t.Fatalf("read document status: %v", err)
			}
			if status == "published" {
				t.Fatalf("illegal transition draft->published was not blocked")
			}
		}
		return
	}

	if !strings.Contains(err.Error(), "42501") &&
		!strings.Contains(err.Error(), "insufficient_privilege") &&
		!strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("unexpected error (wanted privilege error): %v", err)
	}
}

// TestIllegalTransitionBlocked verifies the legal-transition trigger blocks
// direct status updates that skip the allowed state machine.
func TestIllegalTransitionBlocked(t *testing.T) {
	ctx := context.Background()
	db, schema := testdb.Open(t)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	tenantID := testdb.DeterministicID(t, "tenant")
	docID := testdb.DeterministicID(t, "doc")
	userID := testdb.DeterministicID(t, "user")

	fixtures.SeedTenant(t, ctx, db, schema, tenantID)
	fixtures.SeedUser(t, ctx, db, schema, userID, "Trigger Test User")
	fixtures.SeedDocument(t, ctx, db, schema, docID, tenantID, userID)

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET status = 'published'
		 WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		testdb.Qualified(schema, "documents")),
		docID, tenantID,
	)
	if err != nil {
		t.Logf("PASS: illegal transition blocked with error: %v", err)
		return
	}

	var status string
	rowErr := tx.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT status FROM %s WHERE id = $1::uuid`,
		testdb.Qualified(schema, "documents"),
	), docID).Scan(&status)
	if rowErr != nil && rowErr != sql.ErrNoRows {
		t.Fatalf("read status: %v", rowErr)
	}
	if status == "published" {
		t.Fatalf("illegal transition draft->published was not blocked by trigger")
	}
}
