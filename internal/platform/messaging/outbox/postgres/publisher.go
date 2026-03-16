package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/platform/messaging"
)

type Publisher struct {
	db *sql.DB
}

func NewPublisher(db *sql.DB) *Publisher {
	return &Publisher{db: db}
}

func (p *Publisher) Publish(ctx context.Context, event messaging.Event) error {
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	const q = `
INSERT INTO metaldocs.outbox_events (
  event_id, event_type, aggregate_type, aggregate_id, occurred_at, version,
  idempotency_key, producer, trace_id, payload, published_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, NULL)
ON CONFLICT (idempotency_key) DO NOTHING
`
	occurredAt := time.Now().UTC()
	if raw := strings.TrimSpace(event.OccurredAtRFC3339); raw != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
			occurredAt = parsed.UTC()
		}
	}

	if _, err := p.db.ExecContext(
		ctx,
		q,
		strings.TrimSpace(event.EventID),
		strings.TrimSpace(event.EventType),
		strings.TrimSpace(event.AggregateType),
		strings.TrimSpace(event.AggregateID),
		occurredAt,
		event.Version,
		strings.TrimSpace(event.IdempotencyKey),
		strings.TrimSpace(event.Producer),
		strings.TrimSpace(event.TraceID),
		string(payloadJSON),
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}
