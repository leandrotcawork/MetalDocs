package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/iam/domain"
)

// DevRoleProvider is a deterministic in-memory provider used for local memory mode.
type DevRoleProvider struct {
	rolesByUser map[string][]domain.Role
}

func NewDevRoleProvider(rolesByUser map[string][]domain.Role) *DevRoleProvider {
	if rolesByUser == nil {
		rolesByUser = map[string][]domain.Role{}
	}
	return &DevRoleProvider{rolesByUser: rolesByUser}
}

func (p *DevRoleProvider) RolesByUserID(_ context.Context, userID string) ([]domain.Role, error) {
	id := strings.TrimSpace(userID)
	if id == "" {
		return nil, domain.ErrUserNotFound
	}
	roles, ok := p.rolesByUser[id]
	if !ok || len(roles) == 0 {
		return nil, domain.ErrUserNotFound
	}
	out := make([]domain.Role, len(roles))
	copy(out, roles)
	return out, nil
}
