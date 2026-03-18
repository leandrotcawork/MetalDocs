package memory

import (
	"context"
	"sync"
	"time"

	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
)

type Repository struct {
	mu    sync.Mutex
	items []notificationdomain.Notification
	keys  map[string]struct{}
}

func NewRepository() *Repository {
	return &Repository{items: []notificationdomain.Notification{}, keys: map[string]struct{}{}}
}

func (r *Repository) Create(_ context.Context, notification notificationdomain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.keys[notification.IdempotencyKey]; exists {
		return nil
	}
	r.keys[notification.IdempotencyKey] = struct{}{}
	r.items = append(r.items, notification)
	return nil
}

func (r *Repository) List(_ context.Context, query notificationdomain.ListNotificationsQuery) ([]notificationdomain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	out := make([]notificationdomain.Notification, 0, len(r.items))
	for i := len(r.items) - 1; i >= 0; i-- {
		item := r.items[i]
		if query.RecipientUserID != "" && item.RecipientUserID != query.RecipientUserID {
			continue
		}
		if query.Status != "" && item.Status != query.Status {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (r *Repository) MarkRead(_ context.Context, notificationID, recipientUserID string, readAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.items {
		if r.items[i].ID != notificationID || r.items[i].RecipientUserID != recipientUserID {
			continue
		}
		r.items[i].Status = notificationdomain.StatusRead
		readAtUTC := readAt.UTC()
		r.items[i].ReadAt = &readAtUTC
		return nil
	}
	return notificationapp.ErrNotificationNotFound
}

func (r *Repository) Items() []notificationdomain.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]notificationdomain.Notification, len(r.items))
	copy(out, r.items)
	return out
}
