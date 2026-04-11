package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"metaldocs/internal/modules/documents/domain"
)

type ShadowDiffRepository struct {
	db *sql.DB
}

func NewShadowDiffRepository(db *sql.DB) *ShadowDiffRepository {
	return &ShadowDiffRepository{db: db}
}

func (r *ShadowDiffRepository) Insert(ctx context.Context, event domain.ShadowDiffEvent) error {
	summaryBytes, err := json.Marshal(event.DiffSummary)
	if err != nil {
		return fmt.Errorf("marshal diff summary: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO metaldocs.mddm_shadow_diff_events (
			document_id, version_number, user_id_hash,
			current_xml_hash, shadow_xml_hash, diff_summary,
			current_duration_ms, shadow_duration_ms, shadow_error,
			recorded_at, trace_id
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, NULLIF($9, ''), $10, NULLIF($11, ''))`,
		event.DocumentID, event.VersionNumber, event.UserIDHash,
		event.CurrentXMLHash, event.ShadowXMLHash, string(summaryBytes),
		event.CurrentDurationMs, event.ShadowDurationMs, event.ShadowError,
		event.RecordedAt, event.TraceID)
	if err != nil {
		return fmt.Errorf("insert shadow diff event: %w", err)
	}
	return nil
}
