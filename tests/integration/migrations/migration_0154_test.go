//go:build integration

package migrations_test

import (
	"context"
	"os"
	"testing"

	"metaldocs/tests/integration/testdb"
)

func TestMigration0154_FileExists(t *testing.T) {
	if _, err := os.Stat("../../../migrations/0154_capability_doc_edit_draft.sql"); err != nil {
		t.Fatalf("expected migration file 0154_capability_doc_edit_draft.sql: %v", err)
	}
}

func TestMigration0154_DocEditDraftCapabilitySeeded(t *testing.T) {
	ctx := context.Background()
	db, _ := testdb.Open(t)

	roles := []string{"author", "qms_admin"}
	for _, role := range roles {
		var found bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				  FROM metaldocs.role_capabilities
				 WHERE capability = 'doc.edit_draft'
				   AND role = $1
			)`, role).Scan(&found)
		if err != nil {
			t.Fatalf("query role_capabilities for role=%s: %v", role, err)
		}
		if !found {
			t.Fatalf("missing capability doc.edit_draft for role=%s", role)
		}
	}
}
