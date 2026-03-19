package memory

import (
	"context"
	"sync"

	"metaldocs/internal/modules/iam/domain"
)

type RoleAdminRepository struct {
	mu    sync.Mutex
	users map[string]userRecord
}

type userRecord struct {
	displayName string
	roles       map[domain.Role]bool
}

func NewRoleAdminRepository() *RoleAdminRepository {
	return &RoleAdminRepository{users: map[string]userRecord{}}
}

func (r *RoleAdminRepository) HasAnyRole(_ context.Context, role domain.Role) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rec := range r.users {
		if rec.roles[role] {
			return true, nil
		}
	}
	return false, nil
}

func (r *RoleAdminRepository) UpsertUserAndAssignRole(_ context.Context, userID, displayName string, role domain.Role, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.users[userID]
	if !ok {
		rec = userRecord{displayName: displayName, roles: map[domain.Role]bool{}}
	}
	rec.displayName = displayName
	rec.roles[role] = true
	r.users[userID] = rec
	return nil
}

func (r *RoleAdminRepository) ReplaceUserRoles(_ context.Context, userID, displayName string, roles []domain.Role, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.users[userID]
	if !ok {
		rec = userRecord{displayName: displayName, roles: map[domain.Role]bool{}}
	}
	rec.displayName = displayName
	rec.roles = map[domain.Role]bool{}
	for _, role := range roles {
		rec.roles[role] = true
	}
	r.users[userID] = rec
	return nil
}
