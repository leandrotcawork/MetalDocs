//go:build integration

package application

import (
	"context"
	"fmt"
	"testing"
	"time"

	"metaldocs/internal/modules/iam/domain"
)

type benchUserAreaRepo struct {
	byUserTenant map[string][]domain.UserProcessArea
}

func (r *benchUserAreaRepo) ListActive(ctx context.Context, userID, tenantID string, now time.Time) ([]domain.UserProcessArea, error) {
	return r.byUserTenant[userID+"|"+tenantID], nil
}

type benchPolicyRepo struct{}

func (r *benchPolicyRepo) ListForUser(ctx context.Context, userID, tenantID, areaCode string) ([]AccessPolicy, error) {
	return nil, nil
}

func BenchmarkAuthzCheck(b *testing.B) {
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	svc := newBenchAuthorizationService(now)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := svc.Check(
			context.Background(),
			"user-01",
			"tenant-01",
			domain.CapDocumentCreate,
			ResourceCtx{AreaCode: "area-01"},
		)
		if err != nil {
			b.Fatalf("check failed: %v", err)
		}
	}
}

func BenchmarkAuthzCheck_Cached(b *testing.B) {
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	svc := newBenchAuthorizationService(now)
	ctx := WithAuthzCache(context.Background())

	if err := svc.Check(
		ctx,
		"user-01",
		"tenant-01",
		domain.CapDocumentCreate,
		ResourceCtx{AreaCode: "area-01"},
	); err != nil {
		b.Fatalf("pre-warm check failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := svc.Check(
			ctx,
			"user-01",
			"tenant-01",
			domain.CapDocumentCreate,
			ResourceCtx{AreaCode: "area-01"},
		)
		if err != nil {
			b.Fatalf("cached check failed: %v", err)
		}
	}
}

func newBenchAuthorizationService(now time.Time) *AuthorizationService {
	userAreas := &benchUserAreaRepo{
		byUserTenant: make(map[string][]domain.UserProcessArea, 10),
	}
	for u := 1; u <= 10; u++ {
		userID := fmt.Sprintf("user-%02d", u)
		key := userID + "|tenant-01"
		memberships := make([]domain.UserProcessArea, 0, 5)
		for a := 1; a <= 5; a++ {
			memberships = append(memberships, domain.UserProcessArea{
				UserID:        userID,
				TenantID:      "tenant-01",
				AreaCode:      fmt.Sprintf("area-%02d", a),
				Role:          domain.RoleEditor,
				EffectiveFrom: now.Add(-time.Hour),
			})
		}
		userAreas.byUserTenant[key] = memberships
	}

	svc := NewAuthorizationService(userAreas, &benchPolicyRepo{}, nil)
	svc.nowFn = func() time.Time { return now }
	return svc
}
