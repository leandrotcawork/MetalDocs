package postgres

import (
	"context"
	"database/sql"
	"fmt"

	notificationdomain "metaldocs/internal/modules/notifications/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, notification notificationdomain.Notification) error {
	const q = `
INSERT INTO metaldocs.notifications (
  id, recipient_user_id, event_type, resource_type, resource_id,
  title, message, status, idempotency_key, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (idempotency_key) DO NOTHING
`
	_, err := r.db.ExecContext(ctx, q,
		notification.ID,
		notification.RecipientUserID,
		notification.EventType,
		notification.ResourceType,
		notification.ResourceID,
		notification.Title,
		notification.Message,
		notification.Status,
		notification.IdempotencyKey,
		notification.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}
