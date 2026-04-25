package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/taxonomy/domain"
)

func TestAreaServiceSetParentValid(t *testing.T) {
	repo := newFakeAreaRepository()
	repo.put(&domain.ProcessArea{Code: "root", TenantID: "tenant-a"})
	repo.put(&domain.ProcessArea{Code: "child", TenantID: "tenant-a"})

	service := NewAreaService(repo, &fakeGovernanceLogger{})
	err := service.SetParent(context.Background(), "tenant-a", "child", strPtr("root"), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := repo.get("tenant-a", "child")
	if updated.ParentCode == nil || *updated.ParentCode != "root" {
		t.Fatalf("expected parent code root, got %+v", updated.ParentCode)
	}
}

func TestAreaServiceSetParentFailsWhenCycleDetected(t *testing.T) {
	repo := newFakeAreaRepository()
	repo.put(&domain.ProcessArea{Code: "parent", TenantID: "tenant-a"})
	repo.put(&domain.ProcessArea{Code: "child", TenantID: "tenant-a"})
	repo.ancestorsByCode["parent"] = []string{"child"}

	service := NewAreaService(repo, &fakeGovernanceLogger{})
	err := service.SetParent(context.Background(), "tenant-a", "child", strPtr("parent"), "user-1")
	if !errors.Is(err, domain.ErrAreaParentCycle) {
		t.Fatalf("expected ErrAreaParentCycle, got %v", err)
	}
}

func TestAreaServiceArchiveSetsArchivedAt(t *testing.T) {
	repo := newFakeAreaRepository()
	repo.put(&domain.ProcessArea{Code: "child", TenantID: "tenant-a"})

	service := NewAreaService(repo, &fakeGovernanceLogger{})
	fixedNow := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixedNow }

	err := service.Archive(context.Background(), "tenant-a", "child", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := repo.get("tenant-a", "child")
	if updated.ArchivedAt == nil || !updated.ArchivedAt.Equal(fixedNow) {
		t.Fatalf("expected archived_at %s, got %+v", fixedNow, updated.ArchivedAt)
	}
}

func TestAreaServiceArchiveFailsWhenAlreadyArchived(t *testing.T) {
	archivedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	repo := newFakeAreaRepository()
	repo.put(&domain.ProcessArea{Code: "child", TenantID: "tenant-a", ArchivedAt: &archivedAt})

	service := NewAreaService(repo, &fakeGovernanceLogger{})
	err := service.Archive(context.Background(), "tenant-a", "child", "user-1")
	if !errors.Is(err, domain.ErrAreaArchived) {
		t.Fatalf("expected ErrAreaArchived, got %v", err)
	}
}

type fakeAreaRepository struct {
	byKey           map[string]*domain.ProcessArea
	ancestorsByCode map[string][]string
}

func newFakeAreaRepository() *fakeAreaRepository {
	return &fakeAreaRepository{
		byKey:           map[string]*domain.ProcessArea{},
		ancestorsByCode: map[string][]string{},
	}
}

func (r *fakeAreaRepository) GetByCode(_ context.Context, tenantID, code string) (*domain.ProcessArea, error) {
	item, ok := r.byKey[tenantID+"|"+code]
	if !ok {
		return nil, domain.ErrAreaNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *fakeAreaRepository) List(_ context.Context, _ string, _ bool) ([]domain.ProcessArea, error) {
	return nil, nil
}

func (r *fakeAreaRepository) Create(_ context.Context, a *domain.ProcessArea) error {
	r.put(a)
	return nil
}

func (r *fakeAreaRepository) Update(_ context.Context, a *domain.ProcessArea) error {
	r.put(a)
	return nil
}

func (r *fakeAreaRepository) ListAncestors(_ context.Context, _ string, code string) ([]string, error) {
	return r.ancestorsByCode[code], nil
}

func (r *fakeAreaRepository) put(a *domain.ProcessArea) {
	copy := *a
	r.byKey[a.TenantID+"|"+a.Code] = &copy
}

func (r *fakeAreaRepository) get(tenantID, code string) *domain.ProcessArea {
	return r.byKey[tenantID+"|"+code]
}

func strPtr(v string) *string {
	return &v
}
