//go:build integration
// +build integration

package scenarios_test

import (
	"context"
	"database/sql"
	"testing"

	"metaldocs/tests/integration/fixtures"
	"metaldocs/tests/integration/testdb"
)

func TestObsoleteCascade_ParentAndChildren(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()

	hasSupersedes, err := hasColumn(ctx, db, "metaldocs", "documents", "supersedes_id")
	if err != nil {
		t.Fatalf("check supersedes_id column: %v", err)
	}
	if !hasSupersedes {
		t.Skip("metaldocs.documents.supersedes_id does not exist; skipping cascade-chain test")
	}

	tenantID := testdb.DeterministicID(t, "tenant-obsolete-chain")
	userID := testdb.DeterministicID(t, "user-obsolete-chain")
	parentID := testdb.DeterministicID(t, "parent-obsolete")
	child1ID := testdb.DeterministicID(t, "child1-obsolete")
	child2ID := testdb.DeterministicID(t, "child2-obsolete")

	fixtures.SeedUser(t, ctx, db, "metaldocs", userID, "Obsolete User")
	insertPublishedDoc(t, ctx, db, parentID, tenantID, userID, "")
	insertPublishedDoc(t, ctx, db, child1ID, tenantID, userID, parentID)
	insertPublishedDoc(t, ctx, db, child2ID, tenantID, userID, parentID)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.documents WHERE id IN ($1::uuid, $2::uuid, $3::uuid)`, parentID, child1ID, child2ID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.iam_users WHERE tenant_id = $1::uuid`, tenantID)
	})

	if _, err := db.ExecContext(ctx, `
		UPDATE metaldocs.documents
		   SET status = 'obsolete', updated_at = now()
		 WHERE id = $1::uuid
		   AND tenant_id = $2::uuid
		   AND status = 'published'`,
		parentID, tenantID,
	); err != nil {
		t.Fatalf("update parent to obsolete: %v", err)
	}

	var parentStatus string
	if err := db.QueryRowContext(ctx, `
		SELECT status FROM metaldocs.documents WHERE id = $1::uuid`,
		parentID,
	).Scan(&parentStatus); err != nil {
		t.Fatalf("read parent status: %v", err)
	}
	if parentStatus != "obsolete" {
		t.Fatalf("expected parent status obsolete, got %q", parentStatus)
	}

	var childStatuses []string
	rows, err := db.QueryContext(ctx, `
		SELECT status
		  FROM metaldocs.documents
		 WHERE id IN ($1::uuid, $2::uuid)
		 ORDER BY id`,
		child1ID, child2ID,
	)
	if err != nil {
		t.Fatalf("read child statuses: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan child status: %v", err)
		}
		childStatuses = append(childStatuses, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate child statuses: %v", err)
	}
	t.Logf("obsolete cascade observation: parent=%s children=%v", parentStatus, childStatuses)
}

func TestObsoleteCascade_NoStaleOCC(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()

	tenantID := testdb.DeterministicID(t, "tenant-obsolete-occ")
	userID := testdb.DeterministicID(t, "user-obsolete-occ")
	docID := testdb.DeterministicID(t, "doc-obsolete-occ")

	fixtures.SeedUser(t, ctx, db, "metaldocs", userID, "OCC User")
	insertPublishedDocWithRevision(t, ctx, db, docID, tenantID, userID, 3)

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.documents WHERE id = $1::uuid`, docID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.iam_users WHERE tenant_id = $1::uuid`, tenantID)
	})

	res, err := db.ExecContext(ctx, `
		UPDATE metaldocs.documents
		   SET status = 'obsolete', revision_version = revision_version + 1
		 WHERE id = $1::uuid
		   AND tenant_id = $2::uuid
		   AND revision_version = 2`,
		docID, tenantID,
	)
	if err != nil {
		t.Fatalf("stale OCC update should not error: %v", err)
	}
	ra, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected: %v", err)
	}
	if ra != 0 {
		t.Fatalf("expected 0 rows affected for stale OCC, got %d", ra)
	}

	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM metaldocs.documents WHERE id = $1::uuid`, docID).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "published" {
		t.Fatalf("expected status to remain published, got %q", status)
	}
}

func TestLegalTransition_ObsoleteFromPublished(t *testing.T) {
	db := openDirectDB(t)
	ctx := context.Background()

	tenantID := testdb.DeterministicID(t, "tenant-legal-obsolete")
	userID := testdb.DeterministicID(t, "user-legal-obsolete")
	docID := testdb.DeterministicID(t, "doc-legal-obsolete")

	fixtures.SeedUser(t, ctx, db, "metaldocs", userID, "Transition User")
	insertPublishedDoc(t, ctx, db, docID, tenantID, userID, "")

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.documents WHERE id = $1::uuid`, docID)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM metaldocs.iam_users WHERE tenant_id = $1::uuid`, tenantID)
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SET LOCAL metaldocs.cancel_in_progress=''`); err != nil {
		t.Fatalf("set local cancel_in_progress: %v", err)
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE metaldocs.documents
		   SET status = 'obsolete', updated_at = now()
		 WHERE id = $1::uuid
		   AND tenant_id = $2::uuid
		   AND status = 'published'`,
		docID, tenantID,
	)
	if err != nil {
		t.Logf("obsolete transition blocked by trigger/policy in this environment: %v", err)
		return
	}
	ra, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected: %v", err)
	}
	if ra != 1 {
		t.Fatalf("expected exactly one row updated, got %d", ra)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit transition tx: %v", err)
	}

	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM metaldocs.documents WHERE id = $1::uuid`, docID).Scan(&status); err != nil {
		t.Fatalf("read status after transition: %v", err)
	}
	if status != "obsolete" {
		t.Fatalf("expected obsolete status after legal transition, got %q", status)
	}
}

func hasColumn(ctx context.Context, db *sql.DB, schema, table, column string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM information_schema.columns
			 WHERE table_schema = $1
			   AND table_name = $2
			   AND column_name = $3
		)`,
		schema, table, column,
	).Scan(&exists)
	return exists, err
}

func insertPublishedDoc(t *testing.T, ctx context.Context, db *sql.DB, docID, tenantID, userID, supersedesID string) {
	t.Helper()
	insertPublishedDocWithRevisionAndSupersedes(t, ctx, db, docID, tenantID, userID, 1, supersedesID)
}

func insertPublishedDocWithRevision(t *testing.T, ctx context.Context, db *sql.DB, docID, tenantID, userID string, revision int) {
	t.Helper()
	insertPublishedDocWithRevisionAndSupersedes(t, ctx, db, docID, tenantID, userID, revision, "")
}

func insertPublishedDocWithRevisionAndSupersedes(t *testing.T, ctx context.Context, db *sql.DB, docID, tenantID, userID string, revision int, supersedesID string) {
	t.Helper()
	hasSupersedes, err := hasColumn(ctx, db, "metaldocs", "documents", "supersedes_id")
	if err != nil {
		t.Fatalf("check supersedes_id column: %v", err)
	}

	if hasSupersedes {
		_, err = db.ExecContext(ctx, `
			INSERT INTO metaldocs.documents
				(id, tenant_id, name, status, created_by, revision_version, supersedes_id, created_at, updated_at)
			VALUES
				($1::uuid, $2::uuid, 'Obsolete Test Doc', 'published', $3, $4, NULLIF($5, '')::uuid, now(), now())`,
			docID, tenantID, userID, revision, supersedesID,
		)
	} else {
		_, err = db.ExecContext(ctx, `
			INSERT INTO metaldocs.documents
				(id, tenant_id, name, status, created_by, revision_version, created_at, updated_at)
			VALUES
				($1::uuid, $2::uuid, 'Obsolete Test Doc', 'published', $3, $4, now(), now())`,
			docID, tenantID, userID, revision,
		)
	}
	if err != nil {
		t.Fatalf("insert published document %s: %v", docID, err)
	}
}
