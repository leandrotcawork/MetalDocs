package application

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

const registryBackfillAdvisoryLock int64 = 903210421

func BackfillLegacyDocuments(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	var locked bool
	if err := db.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, registryBackfillAdvisoryLock).Scan(&locked); err != nil {
		return fmt.Errorf("acquire backfill advisory lock: %w", err)
	}
	if !locked {
		logger.Info("registry backfill skipped; advisory lock not acquired")
		return nil
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, registryBackfillAdvisoryLock)
	}()

	rows, err := db.QueryContext(ctx, `
		SELECT id::text, tenant_id::text
		FROM documents_v2
		WHERE controlled_document_id IS NULL
		ORDER BY created_at ASC`)
	if err != nil {
		return fmt.Errorf("query legacy documents: %w", err)
	}
	defer rows.Close()

	processed := 0
	for rows.Next() {
		var docID string
		var tenantID string
		if err := rows.Scan(&docID, &tenantID); err != nil {
			return fmt.Errorf("scan legacy document: %w", err)
		}

		legacyCode := "MIG-" + strings.ToUpper(strings.ReplaceAll(docID, "-", "")[:8])
		var controlledDocumentID string
		if err := db.QueryRowContext(ctx, `
			INSERT INTO controlled_documents
				(tenant_id, profile_code, process_area_code, code, sequence_num, title, owner_user_id, status)
			VALUES
				($1, 'unassigned', 'unassigned', $2, NULL, 'Legacy backfill', 'system', 'active')
			ON CONFLICT (tenant_id, profile_code, code)
			DO UPDATE SET updated_at = now()
			RETURNING id::text`,
			tenantID, legacyCode,
		).Scan(&controlledDocumentID); err != nil {
			return fmt.Errorf("upsert controlled document for legacy row %s: %w", docID, err)
		}

		if _, err := db.ExecContext(ctx, `
			UPDATE documents_v2
			SET controlled_document_id = $1,
			    profile_code_snapshot = 'unassigned',
			    process_area_code_snapshot = 'unassigned'
			WHERE id = $2
			  AND controlled_document_id IS NULL`,
			controlledDocumentID, docID,
		); err != nil {
			return fmt.Errorf("backfill document %s: %w", docID, err)
		}
		processed++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy documents: %w", err)
	}

	logger.Info("registry backfill completed", "processed", processed)
	return nil
}
