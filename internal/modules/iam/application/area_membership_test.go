package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

type userAreaWriteRepoStub struct {
	active           *domain.UserProcessArea
	closeCalls       int
	insertCalls      int
	grantAtomicCalls int
	closedAt         time.Time
	inserted         domain.UserProcessArea
	atomicOld        domain.UserProcessArea
	atomicNew        domain.UserProcessArea
	getActiveErr     error
	closeActiveErr   error
	insertErr        error
	grantAtomicErr   error
}

func (s *userAreaWriteRepoStub) Insert(ctx context.Context, membership domain.UserProcessArea) error {
	if s.insertErr != nil {
		return s.insertErr
	}
	s.insertCalls++
	s.inserted = membership
	return nil
}

func (s *userAreaWriteRepoStub) CloseActive(ctx context.Context, userID, tenantID, areaCode string, effectiveTo time.Time) error {
	if s.closeActiveErr != nil {
		return s.closeActiveErr
	}
	s.closeCalls++
	s.closedAt = effectiveTo
	return nil
}

func (s *userAreaWriteRepoStub) GrantAtomic(ctx context.Context, oldMembership, newMembership domain.UserProcessArea) error {
	if s.grantAtomicErr != nil {
		return s.grantAtomicErr
	}
	s.grantAtomicCalls++
	s.atomicOld = oldMembership
	s.atomicNew = newMembership
	return nil
}

func (s *userAreaWriteRepoStub) GetActiveByUserAndArea(ctx context.Context, userID, tenantID, areaCode string, now time.Time) (*domain.UserProcessArea, error) {
	if s.getActiveErr != nil {
		return nil, s.getActiveErr
	}
	return s.active, nil
}

type membershipLoggerStub struct {
	actions []string
}

func (s *membershipLoggerStub) Log(ctx context.Context, action string, membership domain.UserProcessArea) error {
	s.actions = append(s.actions, action)
	return nil
}

func TestGrant_New(t *testing.T) {
	repo := &userAreaWriteRepoStub{}
	logger := &membershipLoggerStub{}
	service := NewAreaMembershipService(repo, logger)
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	service.nowFn = func() time.Time { return now }

	err := service.Grant(context.Background(), "u1", "t1", "A1", domain.RoleEditor, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.closeCalls != 0 {
		t.Fatalf("expected no close call, got %d", repo.closeCalls)
	}
	if repo.grantAtomicCalls != 0 {
		t.Fatalf("expected no atomic grant call, got %d", repo.grantAtomicCalls)
	}
	if repo.insertCalls != 1 {
		t.Fatalf("expected one insert call, got %d", repo.insertCalls)
	}
	if repo.inserted.Role != domain.RoleEditor {
		t.Fatalf("expected inserted role editor, got %q", repo.inserted.Role)
	}
	if repo.inserted.EffectiveFrom != now {
		t.Fatalf("expected effective_from %v, got %v", now, repo.inserted.EffectiveFrom)
	}
	if len(logger.actions) != 1 || logger.actions[0] != "role.grant" {
		t.Fatalf("expected role.grant log, got %v", logger.actions)
	}
}

func TestGrant_Overlap_Merge(t *testing.T) {
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	repo := &userAreaWriteRepoStub{
		active: &domain.UserProcessArea{
			UserID:        "u1",
			TenantID:      "t1",
			AreaCode:      "A1",
			Role:          domain.RoleViewer,
			EffectiveFrom: now.Add(-2 * time.Hour),
		},
	}
	logger := &membershipLoggerStub{}
	service := NewAreaMembershipService(repo, logger)
	service.nowFn = func() time.Time { return now }

	err := service.Grant(context.Background(), "u1", "t1", "A1", domain.RoleReviewer, "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.closeCalls != 0 {
		t.Fatalf("expected no close call (atomic path), got %d", repo.closeCalls)
	}
	if repo.insertCalls != 0 {
		t.Fatalf("expected no direct insert call (atomic path), got %d", repo.insertCalls)
	}
	if repo.grantAtomicCalls != 1 {
		t.Fatalf("expected one atomic grant call, got %d", repo.grantAtomicCalls)
	}
	if repo.atomicNew.Role != domain.RoleReviewer {
		t.Fatalf("expected atomic new role reviewer, got %q", repo.atomicNew.Role)
	}
}

func TestRevoke_Active(t *testing.T) {
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	repo := &userAreaWriteRepoStub{
		active: &domain.UserProcessArea{
			UserID:        "u1",
			TenantID:      "t1",
			AreaCode:      "A1",
			Role:          domain.RoleApprover,
			EffectiveFrom: now.Add(-time.Hour),
		},
	}
	logger := &membershipLoggerStub{}
	service := NewAreaMembershipService(repo, logger)
	service.nowFn = func() time.Time { return now }

	err := service.Revoke(context.Background(), "u1", "t1", "A1", "admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.closeCalls != 1 {
		t.Fatalf("expected one close call, got %d", repo.closeCalls)
	}
	if len(logger.actions) != 1 || logger.actions[0] != "role.revoke" {
		t.Fatalf("expected role.revoke log, got %v", logger.actions)
	}
}

func TestRevoke_NonExistent(t *testing.T) {
	repo := &userAreaWriteRepoStub{}
	service := NewAreaMembershipService(repo, &membershipLoggerStub{})

	err := service.Revoke(context.Background(), "u1", "t1", "A1", "admin")
	if !errors.Is(err, ErrMembershipNotFound) {
		t.Fatalf("expected ErrMembershipNotFound, got %v", err)
	}
}

func TestGrant_UnknownRole(t *testing.T) {
	repo := &userAreaWriteRepoStub{}
	service := NewAreaMembershipService(repo, &membershipLoggerStub{})

	err := service.Grant(context.Background(), "u1", "t1", "A1", domain.Role("ghost"), "admin")
	if !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("expected ErrUnknownRole, got %v", err)
	}
}

func TestTemporalQuery_EffectiveTo_Past(t *testing.T) {
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	repo := &userAreaWriteRepoStub{
		active: &domain.UserProcessArea{
			UserID:        "u1",
			TenantID:      "t1",
			AreaCode:      "A1",
			Role:          domain.RoleEditor,
			EffectiveFrom: now.Add(-time.Hour),
			EffectiveTo:   &past,
		},
	}
	service := NewAreaMembershipService(repo, &membershipLoggerStub{})
	service.nowFn = func() time.Time { return now }

	err := service.Revoke(context.Background(), "u1", "t1", "A1", "admin")
	if !errors.Is(err, ErrMembershipNotFound) {
		t.Fatalf("expected ErrMembershipNotFound for expired active query result, got %v", err)
	}
}
