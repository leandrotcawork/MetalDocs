package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

type authzUserAreaRepoStub struct {
	items []domain.UserProcessArea
	err   error
}

func (s *authzUserAreaRepoStub) ListActive(ctx context.Context, userID, tenantID string, now time.Time) ([]domain.UserProcessArea, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

type authzPolicyRepoStub struct {
	byArea map[string][]AccessPolicy
	calls  int
}

func (s *authzPolicyRepoStub) ListForUser(ctx context.Context, userID, tenantID, areaCode string) ([]AccessPolicy, error) {
	s.calls++
	return s.byArea[areaCode], nil
}

type authzAuthorCheckerStub struct {
	isAuthor bool
	err      error
	calls    int
}

func (s *authzAuthorCheckerStub) IsAuthor(ctx context.Context, userID, templateVersionID string) (bool, error) {
	s.calls++
	if s.err != nil {
		return false, s.err
	}
	return s.isAuthor, nil
}

func TestAuthz_RoleCapUnion(t *testing.T) {
	now := time.Now().UTC()
	userAreas := &authzUserAreaRepoStub{
		items: []domain.UserProcessArea{
			{UserID: "u1", TenantID: "t1", AreaCode: "Area-A", Role: domain.RoleViewer, EffectiveFrom: now.Add(-time.Hour)},
			{UserID: "u1", TenantID: "t1", AreaCode: "Area-B", Role: domain.RoleEditor, EffectiveFrom: now.Add(-time.Hour)},
		},
	}
	policies := &authzPolicyRepoStub{byArea: map[string][]AccessPolicy{}}
	service := NewAuthorizationService(userAreas, policies, &authzAuthorCheckerStub{})

	allowB, err := service.Check(context.Background(), "u1", "t1", domain.CapDocumentCreate, ResourceCtx{AreaCode: "Area-B"})
	if err != nil {
		t.Fatalf("unexpected error for Area-B: %v", err)
	}
	if !allowB {
		t.Fatalf("expected allow for Area-B")
	}

	allowA, err := service.Check(context.Background(), "u1", "t1", domain.CapDocumentCreate, ResourceCtx{AreaCode: "Area-A"})
	if err != nil {
		t.Fatalf("unexpected error for Area-A: %v", err)
	}
	if allowA {
		t.Fatalf("expected deny for Area-A")
	}
}

func TestAuthz_ExpiredMembership(t *testing.T) {
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	userAreas := &authzUserAreaRepoStub{
		items: []domain.UserProcessArea{
			{
				UserID:        "u1",
				TenantID:      "t1",
				AreaCode:      "Area-A",
				Role:          domain.RoleEditor,
				EffectiveFrom: yesterday.Add(-24 * time.Hour),
				EffectiveTo:   &yesterday,
			},
		},
	}

	service := NewAuthorizationService(userAreas, &authzPolicyRepoStub{byArea: map[string][]AccessPolicy{}}, &authzAuthorCheckerStub{})
	_, err := service.Check(context.Background(), "u1", "t1", domain.CapDocumentEdit, ResourceCtx{AreaCode: "Area-A"})
	if !errors.Is(err, ErrMembershipExpired) {
		t.Fatalf("expected ErrMembershipExpired, got %v", err)
	}
}

func TestAuthz_SoD_TemplateSelfPublish(t *testing.T) {
	now := time.Now().UTC()
	userAreas := &authzUserAreaRepoStub{
		items: []domain.UserProcessArea{
			{UserID: "u1", TenantID: "t1", AreaCode: "Area-A", Role: domain.RoleApprover, EffectiveFrom: now.Add(-time.Hour)},
		},
	}
	authorChecker := &authzAuthorCheckerStub{isAuthor: true}
	service := NewAuthorizationService(userAreas, &authzPolicyRepoStub{byArea: map[string][]AccessPolicy{}}, authorChecker)

	_, err := service.Check(context.Background(), "u1", "t1", domain.CapTemplatePublish, ResourceCtx{
		AreaCode:   "Area-A",
		ResourceID: "tv-1",
	})
	if !errors.Is(err, ErrSoDViolation) {
		t.Fatalf("expected ErrSoDViolation, got %v", err)
	}
	if authorChecker.calls != 1 {
		t.Fatalf("expected IsAuthor to be called once, got %d", authorChecker.calls)
	}
}

func TestAuthz_PerRequestCache(t *testing.T) {
	now := time.Now().UTC()
	userAreas := &authzUserAreaRepoStub{
		items: []domain.UserProcessArea{
			{UserID: "u1", TenantID: "t1", AreaCode: "Area-A", Role: domain.RoleEditor, EffectiveFrom: now.Add(-time.Hour)},
		},
	}
	policies := &authzPolicyRepoStub{byArea: map[string][]AccessPolicy{}}
	service := NewAuthorizationService(userAreas, policies, &authzAuthorCheckerStub{})

	ctx := WithAuthzCache(context.Background())
	allow1, err := service.Check(ctx, "u1", "t1", domain.CapDocumentCreate, ResourceCtx{AreaCode: "Area-A"})
	if err != nil {
		t.Fatalf("unexpected first check error: %v", err)
	}
	allow2, err := service.Check(ctx, "u1", "t1", domain.CapDocumentCreate, ResourceCtx{AreaCode: "Area-A"})
	if err != nil {
		t.Fatalf("unexpected second check error: %v", err)
	}
	if !allow1 || !allow2 {
		t.Fatalf("expected both checks to allow")
	}
	if policies.calls != 1 {
		t.Fatalf("expected one policy repo call due to cache, got %d", policies.calls)
	}
}

func TestAuthz_DenyOverride(t *testing.T) {
	now := time.Now().UTC()
	userAreas := &authzUserAreaRepoStub{
		items: []domain.UserProcessArea{
			{UserID: "u1", TenantID: "t1", AreaCode: "Area-A", Role: domain.RoleEditor, EffectiveFrom: now.Add(-time.Hour)},
		},
	}
	policies := &authzPolicyRepoStub{
		byArea: map[string][]AccessPolicy{
			"Area-A": {
				{Capability: domain.CapDocumentCreate, Effect: "deny"},
			},
		},
	}
	service := NewAuthorizationService(userAreas, policies, &authzAuthorCheckerStub{})
	allow, err := service.Check(context.Background(), "u1", "t1", domain.CapDocumentCreate, ResourceCtx{AreaCode: "Area-A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allow {
		t.Fatalf("expected deny because access policy override is deny")
	}
}

func TestAuthz_AllowOverride(t *testing.T) {
	now := time.Now().UTC()
	userAreas := &authzUserAreaRepoStub{
		items: []domain.UserProcessArea{
			{UserID: "u1", TenantID: "t1", AreaCode: "Area-A", Role: domain.RoleViewer, EffectiveFrom: now.Add(-time.Hour)},
		},
	}
	policies := &authzPolicyRepoStub{
		byArea: map[string][]AccessPolicy{
			"Area-A": {
				{Capability: domain.CapDocumentCreate, Effect: "allow"},
			},
		},
	}
	service := NewAuthorizationService(userAreas, policies, &authzAuthorCheckerStub{})
	allow, err := service.Check(context.Background(), "u1", "t1", domain.CapDocumentCreate, ResourceCtx{AreaCode: "Area-A"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allow {
		t.Fatalf("expected allow because access policy override is allow")
	}
}
