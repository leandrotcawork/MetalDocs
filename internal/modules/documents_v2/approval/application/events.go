package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// GovernanceEvent mirrors the governance_events table columns.
type GovernanceEvent struct {
	TenantID     string
	EventType    string
	ActorUserID  string
	ResourceType string
	ResourceID   string
	Reason       string
	PayloadJSON  json.RawMessage
	OccurredAt   time.Time
}

// EventEmitter writes governance events within the caller's transaction.
// The tx must be the same transaction as the state-change write (outbox pattern).
type EventEmitter interface {
	Emit(ctx context.Context, tx *sql.Tx, event GovernanceEvent) error
}

// sqlEmitter is the default production implementation.
type sqlEmitter struct{}

// NewSQLEmitter returns the production event emitter.
func NewSQLEmitter() EventEmitter { return &sqlEmitter{} }

const insertEventSQL = `
INSERT INTO governance_events
  (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

func (e *sqlEmitter) Emit(ctx context.Context, tx *sql.Tx, ev GovernanceEvent) error {
	payload := ev.PayloadJSON
	if payload == nil {
		payload = json.RawMessage("{}")
	}
	_, err := tx.ExecContext(ctx, insertEventSQL,
		ev.TenantID, ev.EventType, ev.ActorUserID,
		ev.ResourceType, ev.ResourceID, ev.Reason, payload,
	)
	return err
}

// MemoryEmitter is an in-memory EventEmitter for tests.
type MemoryEmitter struct {
	Events []GovernanceEvent
}

func (m *MemoryEmitter) Emit(_ context.Context, _ *sql.Tx, ev GovernanceEvent) error {
	m.Events = append(m.Events, ev)
	return nil
}
