package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

var (
	ErrMembershipExpired = errors.New("membership_expired")
	ErrSoDViolation      = errors.New("sod_violation")
)

type UserAreaRepository interface {
	ListActive(ctx context.Context, userID, tenantID string, now time.Time) ([]domain.UserProcessArea, error)
}

type AccessPolicy struct {
	Capability domain.Capability
	Effect     string
}

type AccessPolicyRepository interface {
	ListForUser(ctx context.Context, userID, tenantID, areaCode string) ([]AccessPolicy, error)
}

type TemplateAuthorChecker interface {
	IsAuthor(ctx context.Context, userID, templateVersionID string) (bool, error)
}

type ResourceCtx struct {
	AreaCode   string
	ResourceID string
}

type AuthorizationService struct {
	userAreas      UserAreaRepository
	accessPolicies AccessPolicyRepository
	authorChecker  TemplateAuthorChecker
	nowFn          func() time.Time
}

func NewAuthorizationService(
	userAreas UserAreaRepository,
	accessPolicies AccessPolicyRepository,
	authorChecker TemplateAuthorChecker,
) *AuthorizationService {
	return &AuthorizationService{
		userAreas:      userAreas,
		accessPolicies: accessPolicies,
		authorChecker:  authorChecker,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

type authzCacheKey struct{}

type authzDecisionKey struct {
	UserID     string
	TenantID   string
	AreaCode   string
	Capability domain.Capability
	ResourceID string
}

func WithAuthzCache(ctx context.Context) context.Context {
	if cache, _ := ctx.Value(authzCacheKey{}).(*sync.Map); cache != nil {
		return ctx
	}
	return context.WithValue(ctx, authzCacheKey{}, &sync.Map{})
}

func (s *AuthorizationService) Check(
	ctx context.Context,
	userID, tenantID string,
	capability domain.Capability,
	resource ResourceCtx,
) (bool, error) {
	cache, _ := ctx.Value(authzCacheKey{}).(*sync.Map)
	key := authzDecisionKey{
		UserID:     strings.TrimSpace(userID),
		TenantID:   strings.TrimSpace(tenantID),
		AreaCode:   strings.TrimSpace(resource.AreaCode),
		Capability: capability,
		ResourceID: strings.TrimSpace(resource.ResourceID),
	}
	if cache != nil {
		if cached, ok := cache.Load(key); ok {
			return cached.(bool), nil
		}
	}

	now := s.nowFn()
	memberships, err := s.userAreas.ListActive(ctx, key.UserID, key.TenantID, now)
	if err != nil {
		return false, fmt.Errorf("list active memberships: %w", err)
	}

	granted := map[domain.Capability]bool{}
	hasMatchingAreaMembership := false
	for _, membership := range memberships {
		if key.AreaCode != "" && membership.AreaCode != key.AreaCode {
			continue
		}
		hasMatchingAreaMembership = true
		if !membership.IsActive(now) {
			return false, ErrMembershipExpired
		}
		for _, cap := range domain.RoleCapabilities[membership.Role] {
			granted[cap] = true
		}
	}

	allowed := granted[capability]
	if hasMatchingAreaMembership {
		policies, err := s.accessPolicies.ListForUser(ctx, key.UserID, key.TenantID, key.AreaCode)
		if err != nil {
			return false, fmt.Errorf("list access policies: %w", err)
		}
		allowOverride := false
		denyOverride := false
		for _, policy := range policies {
			if policy.Capability != capability {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(policy.Effect)) {
			case "deny":
				denyOverride = true
			case "allow":
				allowOverride = true
			}
		}
		if denyOverride {
			allowed = false
		} else if allowOverride {
			allowed = true
		}
	}

	if allowed && capability == domain.CapTemplatePublish && s.authorChecker != nil {
		isAuthor, err := s.authorChecker.IsAuthor(ctx, key.UserID, key.ResourceID)
		if err != nil {
			return false, fmt.Errorf("check template author: %w", err)
		}
		if isAuthor {
			return false, ErrSoDViolation
		}
	}

	if cache != nil {
		cache.Store(key, allowed)
	}
	return allowed, nil
}
