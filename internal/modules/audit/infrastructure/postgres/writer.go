package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"metaldocs/internal/modules/audit/domain"
)

type Writer struct {
	db *sql.DB
}

func NewWriter(db *sql.DB) *Writer {
	return &Writer{db: db}
}

func (w *Writer) Record(ctx context.Context, event domain.Event) error {
	const q = `
INSERT INTO metaldocs.audit_events (
  id, occurred_at, actor_id, action, resource_type, resource_id, payload, trace_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
`
	if _, err := w.db.ExecContext(
		ctx,
		q,
		event.ID,
		event.OccurredAt,
		event.ActorID,
		event.Action,
		event.ResourceType,
		event.ResourceID,
		event.PayloadJSON,
		event.TraceID,
	); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}
