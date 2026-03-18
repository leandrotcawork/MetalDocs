package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	notificationapp "metaldocs/internal/modules/notifications/application"
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

func (r *Repository) List(ctx context.Context, query notificationdomain.ListNotificationsQuery) ([]notificationdomain.Notification, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	base := `
SELECT id, recipient_user_id, event_type, resource_type, resource_id,
       title, message, status, idempotency_key, created_at, read_at
FROM metaldocs.notifications
`
	clauses := []string{}
	args := []any{}
	if strings.TrimSpace(query.RecipientUserID) != "" {
		args = append(args, strings.TrimSpace(query.RecipientUserID))
		clauses = append(clauses, fmt.Sprintf("recipient_user_id = $%d", len(args)))
	}
	if strings.TrimSpace(query.Status) != "" {
		args = append(args, strings.TrimSpace(query.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(clauses) > 0 {
		base += "WHERE " + strings.Join(clauses, " AND ") + "\n"
	}
	args = append(args, limit)
	base += fmt.Sprintf("ORDER BY created_at DESC LIMIT $%d", len(args))

	rows, err := r.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	items := []notificationdomain.Notification{}
	for rows.Next() {
		var item notificationdomain.Notification
		var readAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.RecipientUserID,
			&item.EventType,
			&item.ResourceType,
			&item.ResourceID,
			&item.Title,
			&item.Message,
			&item.Status,
			&item.IdempotencyKey,
			&item.CreatedAt,
			&readAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		if readAt.Valid {
			readAtUTC := readAt.Time.UTC()
			item.ReadAt = &readAtUTC
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return items, nil
}

func (r *Repository) MarkRead(ctx context.Context, notificationID, recipientUserID string, readAt time.Time) error {
	const q = `
UPDATE metaldocs.notifications
SET status = $3,
    read_at = $4
WHERE id = $1
  AND recipient_user_id = $2
`
	result, err := r.db.ExecContext(ctx, q, notificationID, recipientUserID, notificationdomain.StatusRead, readAt.UTC())
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("notification read rows affected: %w", err)
	}
	if rows == 0 {
		return notificationapp.ErrNotificationNotFound
	}
	return nil
}
