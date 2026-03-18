package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

func (w *Writer) ListEvents(ctx context.Context, query domain.ListEventsQuery) ([]domain.Event, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	const q = `
SELECT id, occurred_at, actor_id, action, resource_type, resource_id, payload::text, trace_id
FROM metaldocs.audit_events
WHERE ($1 = '' OR resource_type = $1)
  AND ($2 = '' OR resource_id = $2)
ORDER BY occurred_at DESC, id DESC
LIMIT $3
`

	rows, err := w.db.QueryContext(ctx, q, strings.TrimSpace(query.ResourceType), strings.TrimSpace(query.ResourceID), limit)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Event, 0, limit)
	for rows.Next() {
		var event domain.Event
		if err := rows.Scan(
			&event.ID,
			&event.OccurredAt,
			&event.ActorID,
			&event.Action,
			&event.ResourceType,
			&event.ResourceID,
			&event.PayloadJSON,
			&event.TraceID,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		items = append(items, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return items, nil
}
