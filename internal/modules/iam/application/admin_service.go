package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/iam/domain"
)

type RoleCacheInvalidator interface {
	InvalidateUser(userID string)
}

type AdminService struct {
	repo        domain.RoleAdminRepository
	invalidator RoleCacheInvalidator
}

func NewAdminService(repo domain.RoleAdminRepository, invalidator RoleCacheInvalidator) *AdminService {
	return &AdminService{repo: repo, invalidator: invalidator}
}

func (s *AdminService) UpsertUserAndAssignRole(ctx context.Context, userID, displayName string, role domain.Role, assignedBy string) error {
	userID = strings.TrimSpace(userID)
	displayName = strings.TrimSpace(displayName)
	assignedBy = strings.TrimSpace(assignedBy)

	if userID == "" {
		return domain.ErrUserNotFound
	}
	if displayName == "" {
		displayName = userID
	}
	if assignedBy == "" {
		assignedBy = "system"
	}

	if err := s.repo.UpsertUserAndAssignRole(ctx, userID, displayName, role, assignedBy); err != nil {
		return err
	}
	if s.invalidator != nil {
		s.invalidator.InvalidateUser(userID)
	}
	return nil
}
