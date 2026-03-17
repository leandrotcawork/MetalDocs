package memory

import (
	"context"
	"sync"

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

func (r *Repository) Items() []notificationdomain.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]notificationdomain.Notification, len(r.items))
	copy(out, r.items)
	return out
}
