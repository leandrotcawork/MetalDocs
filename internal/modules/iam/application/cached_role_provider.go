package application

import (
	"context"
	"sync"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

type cacheEntry struct {
	roles     []domain.Role
	expiresAt time.Time
}

// CachedRoleProvider wraps a RoleProvider with TTL cache and explicit invalidation.
type CachedRoleProvider struct {
	base  domain.RoleProvider
	ttl   time.Duration
	mu    sync.RWMutex
	items map[string]cacheEntry
}

func NewCachedRoleProvider(base domain.RoleProvider, ttl time.Duration) *CachedRoleProvider {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &CachedRoleProvider{
		base:  base,
		ttl:   ttl,
		items: map[string]cacheEntry{},
	}
}

func (c *CachedRoleProvider) RolesByUserID(ctx context.Context, userID string) ([]domain.Role, error) {
	now := time.Now().UTC()

	c.mu.RLock()
	entry, ok := c.items[userID]
	c.mu.RUnlock()

	if ok && now.Before(entry.expiresAt) {
		return cloneRoles(entry.roles), nil
	}

	roles, err := c.base.RolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.items[userID] = cacheEntry{roles: cloneRoles(roles), expiresAt: now.Add(c.ttl)}
	c.mu.Unlock()

	return roles, nil
}

func (c *CachedRoleProvider) InvalidateUser(userID string) {
	c.mu.Lock()
	delete(c.items, userID)
	c.mu.Unlock()
}

func (c *CachedRoleProvider) InvalidateAll() {
	c.mu.Lock()
	c.items = map[string]cacheEntry{}
	c.mu.Unlock()
}

func cloneRoles(in []domain.Role) []domain.Role {
	out := make([]domain.Role, len(in))
	copy(out, in)
	return out
}
