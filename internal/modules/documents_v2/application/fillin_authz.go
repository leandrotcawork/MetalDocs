package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"metaldocs/internal/modules/iam/authz"
)

// requireDocEditDraft opens a short tx, sets authz GUCs, resolves the
// document's area_code, and calls authz.Require for "doc.edit_draft".
// The tx is rolled back after the check — no writes happen here.
// If the document doesn't exist, area "tenant" is used (matches submit_service).
func requireDocEditDraft(ctx context.Context, db *sql.DB, tenantID, actorID, docID string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("fillin authz: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ctx = authz.WithCapCache(ctx)

	if err := setAuthzGUC(ctx, tx, tenantID, actorID); err != nil {
		return err
	}
	areaCode, err := loadDocumentAreaCode(ctx, tx, tenantID, docID)
	if err != nil {
		return fmt.Errorf("fillin authz: load area: %w", err)
	}
	return authz.Require(ctx, tx, "doc.edit_draft", areaCode)
}

func setAuthzGUC(ctx context.Context, tx *sql.Tx, tenantID, actorID string) error {
	if _, err := tx.ExecContext(ctx, "SELECT set_config('metaldocs.tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant GUC: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "SELECT set_config('metaldocs.actor_id', $1, true)", actorID); err != nil {
		return fmt.Errorf("set actor GUC: %w", err)
	}
	return nil
}

func loadDocumentAreaCode(ctx context.Context, tx *sql.Tx, tenantID, documentID string) (string, error) {
	var areaCode string
	err := tx.QueryRowContext(ctx, `
		SELECT process_area_code_snapshot
		  FROM documents
		 WHERE id = $1 AND tenant_id = $2`,
		documentID, tenantID,
	).Scan(&areaCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "tenant", nil
		}
		return "", err
	}
	if areaCode == "" {
		return "tenant", nil
	}
	return areaCode, nil
}
