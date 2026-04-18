package application_test

import (
	"context"
	"strconv"
	"testing"

	"metaldocs/internal/modules/templates/application"
	"metaldocs/internal/modules/templates/domain"
)

type fakeRepo struct {
	templates map[string]*domain.Template
	versions  map[string]*domain.TemplateVersion
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{templates: map[string]*domain.Template{}, versions: map[string]*domain.TemplateVersion{}}
}

func (f *fakeRepo) CreateTemplate(_ context.Context, t *domain.Template) (string, error) {
	t.ID = "tpl-" + t.Key
	f.templates[t.ID] = t
	return t.ID, nil
}
func (f *fakeRepo) GetTemplate(_ context.Context, id string) (*domain.Template, error) {
	return f.templates[id], nil
}
func (f *fakeRepo) ListTemplates(_ context.Context, tenantID string) ([]domain.TemplateListItem, error) {
	out := []domain.TemplateListItem{}
	for _, t := range f.templates {
		if t.TenantID != tenantID {
			continue
		}
		latest := 1
		for _, v := range f.versions {
			if v.TemplateID == t.ID && v.VersionNum > latest {
				latest = v.VersionNum
			}
		}
		out = append(out, domain.TemplateListItem{
			ID: t.ID, TenantID: t.TenantID, Key: t.Key, Name: t.Name,
			Description: t.Description, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
			CreatedBy: t.CreatedBy, LatestVersion: latest,
		})
	}
	return out, nil
}
func (f *fakeRepo) CreateVersion(_ context.Context, v *domain.TemplateVersion) (string, error) {
	v.ID = v.TemplateID + "-v" + strconv.Itoa(v.VersionNum)
	f.versions[v.ID] = v
	return v.ID, nil
}
func (f *fakeRepo) GetVersionByNum(_ context.Context, templateID string, n int) (*domain.TemplateVersion, error) {
	for _, v := range f.versions {
		if v.TemplateID == templateID && v.VersionNum == n {
			return v, nil
		}
	}
	return nil, domain.ErrInvalidStateTransition
}
func (f *fakeRepo) UpdateDraftVersion(_ context.Context, v *domain.TemplateVersion, expected int) error {
	cur := f.versions[v.ID]
	if cur.LockVersion != expected {
		return domain.ErrLockVersionMismatch
	}
	cur.LockVersion++
	return nil
}
func (f *fakeRepo) PublishVersion(_ context.Context, id, by string) (string, int, error) {
	v := f.versions[id]
	if v.Status != domain.StatusDraft {
		return "", 0, domain.ErrInvalidStateTransition
	}
	v.Status = domain.StatusPublished
	newVer := domain.NewTemplateVersion(v.TemplateID, v.VersionNum+1)
	newVer.ID = v.TemplateID + "-v" + strconv.Itoa(newVer.VersionNum)
	newVer.DocxStorageKey = v.DocxStorageKey
	newVer.SchemaStorageKey = v.SchemaStorageKey
	newVer.DocxContentHash = v.DocxContentHash
	newVer.SchemaContentHash = v.SchemaContentHash
	newVer.CreatedBy = by
	f.versions[newVer.ID] = newVer
	return newVer.ID, newVer.VersionNum, nil
}

func TestService_CreateTemplate_CreatesV1Draft(t *testing.T) {
	svc := application.New(newFakeRepo(), nil, nil)
	tpl, ver, err := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID: "t1", Key: "po", Name: "Purchase Order", CreatedBy: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Key != "po" {
		t.Fatalf("key mismatch")
	}
	if ver.VersionNum != 1 || ver.Status != domain.StatusDraft {
		t.Fatalf("v1 draft expected")
	}
}

func TestService_SaveDraft_OptimisticLockConflict(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, nil, nil)
	_, ver, _ := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID: "t1", Key: "po", Name: "N", CreatedBy: "u1",
	})
	err := svc.SaveDraft(context.Background(), application.SaveDraftCmd{
		VersionID: ver.ID, ExpectedLockVersion: 99, DocxStorageKey: "k", SchemaStorageKey: "s",
		DocxContentHash: "h", SchemaContentHash: "h",
	})
	if err != domain.ErrLockVersionMismatch {
		t.Fatalf("expected lock mismatch, got %v", err)
	}
}
