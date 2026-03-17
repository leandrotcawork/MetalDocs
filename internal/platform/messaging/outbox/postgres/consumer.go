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

type Consumer struct {
	db *sql.DB
}

var claimedAtSentinel = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

func NewConsumer(db *sql.DB) *Consumer {
	return &Consumer{db: db}
}

func (c *Consumer) ClaimUnpublished(ctx context.Context, limit int) ([]messaging.Event, error) {
	if limit <= 0 {
		limit = 20
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim outbox tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const q = `
SELECT event_id, event_type, aggregate_type, aggregate_id, occurred_at, version,
       idempotency_key, producer, trace_id, payload
FROM metaldocs.outbox_events
WHERE published_at IS NULL
ORDER BY occurred_at ASC
FOR UPDATE SKIP LOCKED
LIMIT $1
`
	rows, err := tx.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("claim unpublished outbox events: %w", err)
	}
	defer rows.Close()

	var events []messaging.Event
	for rows.Next() {
		var event messaging.Event
		var occurredAt time.Time
		var payloadJSON []byte
		if err := rows.Scan(
			&event.EventID,
			&event.EventType,
			&event.AggregateType,
			&event.AggregateID,
			&occurredAt,
			&event.Version,
			&event.IdempotencyKey,
			&event.Producer,
			&event.TraceID,
			&payloadJSON,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		event.OccurredAtRFC3339 = occurredAt.UTC().Format(time.RFC3339)
		if len(payloadJSON) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadJSON, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal outbox payload: %w", err)
			}
			event.Payload = payload
		} else {
			event.Payload = map[string]any{}
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}

	if len(events) > 0 {
		ids := make([]string, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.EventID)
		}
		if err := updatePublishedAtTx(ctx, tx, ids, claimedAtSentinel); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim outbox tx: %w", err)
	}
	return events, nil
}

func (c *Consumer) MarkPublished(ctx context.Context, eventIDs []string) error {
	return updatePublishedAt(ctx, c.db, eventIDs, time.Now().UTC())
}

func (c *Consumer) Release(ctx context.Context, eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(eventIDs))
	args := make([]any, 0, len(eventIDs))
	for idx, eventID := range eventIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx+1))
		args = append(args, strings.TrimSpace(eventID))
	}
	q := fmt.Sprintf("UPDATE metaldocs.outbox_events SET published_at = NULL WHERE event_id IN (%s) AND published_at = $%d", strings.Join(placeholders, ", "), len(args)+1)
	args = append(args, claimedAtSentinel)
	if _, err := c.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("release outbox events: %w", err)
	}
	return nil
}

func updatePublishedAt(ctx context.Context, db *sql.DB, eventIDs []string, publishedAt time.Time) error {
	if len(eventIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(eventIDs))
	args := make([]any, 0, len(eventIDs))
	for idx, eventID := range eventIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx+1))
		args = append(args, strings.TrimSpace(eventID))
	}
	q := fmt.Sprintf("UPDATE metaldocs.outbox_events SET published_at = $%d WHERE event_id IN (%s)", len(args)+1, strings.Join(placeholders, ", "))
	args = append(args, publishedAt.UTC())
	if _, err := db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("mark outbox events published: %w", err)
	}
	return nil
}

func updatePublishedAtTx(ctx context.Context, tx *sql.Tx, eventIDs []string, publishedAt time.Time) error {
	if len(eventIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(eventIDs))
	args := make([]any, 0, len(eventIDs))
	for idx, eventID := range eventIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", idx+1))
		args = append(args, strings.TrimSpace(eventID))
	}
	q := fmt.Sprintf("UPDATE metaldocs.outbox_events SET published_at = $%d WHERE event_id IN (%s)", len(args)+1, strings.Join(placeholders, ", "))
	args = append(args, publishedAt.UTC())
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}
	return nil
}
