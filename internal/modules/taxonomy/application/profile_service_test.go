package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/taxonomy/domain"
)

func TestProfileServiceSetDefaultTemplateHappyPath(t *testing.T) {
	repo := newFakeProfileRepository()
	profile := &domain.DocumentProfile{Code: "po", TenantID: "tenant-a"}
	repo.put(profile)

	checker := &fakeTemplateVersionChecker{published: true, profileCode: "po"}
	logger := &fakeGovernanceLogger{}
	service := NewProfileService(repo, checker, logger)

	err := service.SetDefaultTemplate(context.Background(), "tenant-a", "po", "tpl-v1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := repo.get("tenant-a", "po")
	if updated.DefaultTemplateVersionID == nil || *updated.DefaultTemplateVersionID != "tpl-v1" {
		t.Fatalf("expected default template to be tpl-v1, got %+v", updated.DefaultTemplateVersionID)
	}
	if len(logger.events) != 1 {
		t.Fatalf("expected 1 governance event, got %d", len(logger.events))
	}
}

func TestProfileServiceSetDefaultTemplateFailsWhenTemplateNotPublished(t *testing.T) {
	repo := newFakeProfileRepository()
	repo.put(&domain.DocumentProfile{Code: "po", TenantID: "tenant-a"})

	checker := &fakeTemplateVersionChecker{published: false, profileCode: "po"}
	service := NewProfileService(repo, checker, &fakeGovernanceLogger{})

	err := service.SetDefaultTemplate(context.Background(), "tenant-a", "po", "tpl-v1", "user-1")
	if !errors.Is(err, domain.ErrTemplateNotPublished) {
		t.Fatalf("expected ErrTemplateNotPublished, got %v", err)
	}
}

func TestProfileServiceSetDefaultTemplateFailsWhenTemplateBelongsToDifferentProfile(t *testing.T) {
	repo := newFakeProfileRepository()
	repo.put(&domain.DocumentProfile{Code: "po", TenantID: "tenant-a"})

	checker := &fakeTemplateVersionChecker{published: true, profileCode: "it"}
	service := NewProfileService(repo, checker, &fakeGovernanceLogger{})

	err := service.SetDefaultTemplate(context.Background(), "tenant-a", "po", "tpl-v1", "user-1")
	if !errors.Is(err, domain.ErrTemplateProfileMismatch) {
		t.Fatalf("expected ErrTemplateProfileMismatch, got %v", err)
	}
}

func TestProfileServiceArchiveSetsArchivedAt(t *testing.T) {
	repo := newFakeProfileRepository()
	repo.put(&domain.DocumentProfile{Code: "po", TenantID: "tenant-a"})

	service := NewProfileService(repo, &fakeTemplateVersionChecker{}, &fakeGovernanceLogger{})
	fixedNow := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixedNow }

	err := service.Archive(context.Background(), "tenant-a", "po", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated := repo.get("tenant-a", "po")
	if updated.ArchivedAt == nil || !updated.ArchivedAt.Equal(fixedNow) {
		t.Fatalf("expected archived_at to be %s, got %+v", fixedNow, updated.ArchivedAt)
	}
}

func TestProfileServiceArchiveFailsWhenAlreadyArchived(t *testing.T) {
	archivedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	repo := newFakeProfileRepository()
	repo.put(&domain.DocumentProfile{Code: "po", TenantID: "tenant-a", ArchivedAt: &archivedAt})

	service := NewProfileService(repo, &fakeTemplateVersionChecker{}, &fakeGovernanceLogger{})
	err := service.Archive(context.Background(), "tenant-a", "po", "user-1")
	if !errors.Is(err, domain.ErrProfileArchived) {
		t.Fatalf("expected ErrProfileArchived, got %v", err)
	}
}

type fakeProfileRepository struct {
	byKey map[string]*domain.DocumentProfile
}

func newFakeProfileRepository() *fakeProfileRepository {
	return &fakeProfileRepository{byKey: map[string]*domain.DocumentProfile{}}
}

func (r *fakeProfileRepository) GetByCode(_ context.Context, tenantID, code string) (*domain.DocumentProfile, error) {
	item, ok := r.byKey[tenantID+"|"+code]
	if !ok {
		return nil, domain.ErrProfileNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *fakeProfileRepository) List(_ context.Context, _ string, _ bool) ([]domain.DocumentProfile, error) {
	return nil, nil
}

func (r *fakeProfileRepository) Create(_ context.Context, p *domain.DocumentProfile) error {
	r.put(p)
	return nil
}

func (r *fakeProfileRepository) Update(_ context.Context, p *domain.DocumentProfile) error {
	r.put(p)
	return nil
}

func (r *fakeProfileRepository) put(p *domain.DocumentProfile) {
	copy := *p
	r.byKey[p.TenantID+"|"+p.Code] = &copy
}

func (r *fakeProfileRepository) get(tenantID, code string) *domain.DocumentProfile {
	return r.byKey[tenantID+"|"+code]
}

type fakeTemplateVersionChecker struct {
	published   bool
	profileCode string
	err         error
}

func (f *fakeTemplateVersionChecker) IsPublished(_ context.Context, _ string) (bool, string, error) {
	return f.published, f.profileCode, f.err
}

type fakeGovernanceLogger struct {
	events []domain.GovernanceEvent
}

func (f *fakeGovernanceLogger) Log(_ context.Context, e domain.GovernanceEvent) error {
	f.events = append(f.events, e)
	return nil
}
