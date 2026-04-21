package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

var (
	ErrMembershipNotFound = errors.New("membership_not_found")
	ErrUnknownRole        = errors.New("unknown_role")
)

type UserAreaWriteRepository interface {
	Insert(ctx context.Context, membership domain.UserProcessArea) error
	CloseActive(ctx context.Context, userID, tenantID, areaCode string, effectiveTo time.Time) error
	GetActiveByUserAndArea(ctx context.Context, userID, tenantID, areaCode string, now time.Time) (*domain.UserProcessArea, error)
}

type MembershipGovernanceLogger interface {
	Log(ctx context.Context, action string, membership domain.UserProcessArea) error
}

type AreaMembershipService struct {
	repo   UserAreaWriteRepository
	logger MembershipGovernanceLogger
	nowFn  func() time.Time
}

func NewAreaMembershipService(repo UserAreaWriteRepository, logger MembershipGovernanceLogger) *AreaMembershipService {
	return &AreaMembershipService{
		repo:   repo,
		logger: logger,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *AreaMembershipService) Grant(
	ctx context.Context,
	userID, tenantID, areaCode string,
	role domain.Role,
	grantedBy string,
) error {
	if _, ok := domain.RoleCapabilities[role]; !ok {
		return ErrUnknownRole
	}

	now := s.nowFn()
	existing, err := s.repo.GetActiveByUserAndArea(ctx, userID, tenantID, areaCode, now)
	if err != nil {
		return fmt.Errorf("get active membership: %w", err)
	}
	if existing != nil && existing.IsActive(now) {
		if err := s.repo.CloseActive(ctx, userID, tenantID, areaCode, now); err != nil {
			return fmt.Errorf("close active membership: %w", err)
		}
	}

	membership := domain.UserProcessArea{
		UserID:        userID,
		TenantID:      tenantID,
		AreaCode:      areaCode,
		Role:          role,
		EffectiveFrom: now,
	}
	if grantedBy != "" {
		membership.GrantedBy = &grantedBy
	}

	if err := s.repo.Insert(ctx, membership); err != nil {
		return fmt.Errorf("insert membership: %w", err)
	}
	if s.logger != nil {
		if err := s.logger.Log(ctx, "role.grant", membership); err != nil {
			return fmt.Errorf("log membership grant: %w", err)
		}
	}
	return nil
}

func (s *AreaMembershipService) Revoke(
	ctx context.Context,
	userID, tenantID, areaCode string,
	revokedBy string,
) error {
	now := s.nowFn()
	active, err := s.repo.GetActiveByUserAndArea(ctx, userID, tenantID, areaCode, now)
	if err != nil {
		return fmt.Errorf("get active membership: %w", err)
	}
	if active == nil || !active.IsActive(now) {
		return ErrMembershipNotFound
	}

	if err := s.repo.CloseActive(ctx, userID, tenantID, areaCode, now); err != nil {
		return fmt.Errorf("close active membership: %w", err)
	}

	if s.logger != nil {
		membership := *active
		if revokedBy != "" {
			membership.GrantedBy = &revokedBy
		}
		if err := s.logger.Log(ctx, "role.revoke", membership); err != nil {
			return fmt.Errorf("log membership revoke: %w", err)
		}
	}
	return nil
}
