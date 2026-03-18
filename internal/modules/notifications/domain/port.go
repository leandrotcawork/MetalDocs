package domain

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, notification Notification) error
	List(ctx context.Context, query ListNotificationsQuery) ([]Notification, error)
	MarkRead(ctx context.Context, notificationID, recipientUserID string, readAt time.Time) error
}
