package application

import (
	"context"
	"database/sql"

	"metaldocs/internal/modules/taxonomy/domain"
)

type DBGovernanceLogger struct {
	db *sql.DB
}

func NewDBGovernanceLogger(db *sql.DB) *DBGovernanceLogger {
	return &DBGovernanceLogger{db: db}
}

func (l *DBGovernanceLogger) Log(ctx context.Context, e domain.GovernanceEvent) error {
	payload := e.PayloadJSON
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := l.db.ExecContext(
		ctx,
		`INSERT INTO governance_events
		    (tenant_id, event_type, actor_user_id, resource_type, resource_id, reason, payload_json)
		 VALUES
		    ($1, $2, $3, $4, $5, $6, $7)`,
		e.TenantID,
		e.EventType,
		e.ActorUserID,
		e.ResourceType,
		e.ResourceID,
		nullString(e.Reason),
		payload,
	)
	return err
}

func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
