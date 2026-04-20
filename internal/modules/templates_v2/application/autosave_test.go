package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func TestPresignAutosave_Happy(t *testing.T) {
	repo := newFakeRepo()
	template := &domain.Template{
		ID:       "tpl-1",
		TenantID: "tenant-a",
	}
	version := &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  3,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/3.docx",
	}
	repo.templates[template.ID] = template
	repo.versions[version.ID] = version

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.PresignAutosave(context.Background(), application.PresignAutosaveCmd{
		TenantID:      "tenant-a",
		ActorUserID:   "user-a",
		TemplateID:    "tpl-1",
		VersionNumber: 3,
	})
	if err != nil {
		t.Fatalf("PresignAutosave returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if got.UploadURL != "https://presigned/put/templates/tpl-1/versions/3.docx" {
		t.Fatalf("unexpected upload url: %q", got.UploadURL)
	}
	if got.StorageKey != "templates/tpl-1/versions/3.docx" {
		t.Fatalf("unexpected storage key: %q", got.StorageKey)
	}
	wantExpiresAt := time.Date(2026, 4, 20, 12, 10, 0, 0, time.UTC)
	if !got.ExpiresAt.Equal(wantExpiresAt) {
		t.Fatalf("expected expiresAt %s, got %s", wantExpiresAt, got.ExpiresAt)
	}
}

func TestPresignAutosave_NonDraft(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{
		ID:       "tpl-1",
		TenantID: "tenant-a",
	}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  1,
		Status:         domain.VersionStatusInReview,
		DocxStorageKey: "templates/tpl-1/versions/1.docx",
	}

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.PresignAutosave(context.Background(), application.PresignAutosaveCmd{
		TenantID:      "tenant-a",
		ActorUserID:   "user-a",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
	})
	if !errors.Is(err, domain.ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestCommitAutosave_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{
		ID:       "tpl-1",
		TenantID: "tenant-a",
	}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  7,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/7.docx",
	}
	presigner := &fakePresigner{HeadResult: "hash_abc"}
	svc := application.New(repo, presigner, fakeClock{}, &fakeUUID{})

	got, err := svc.CommitAutosave(context.Background(), application.CommitAutosaveCmd{
		TenantID:            "tenant-a",
		ActorUserID:         "user-a",
		TemplateID:          "tpl-1",
		VersionNumber:       7,
		ExpectedContentHash: "hash_abc",
	})
	if err != nil {
		t.Fatalf("CommitAutosave returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil version")
	}
	if got.ContentHash != "hash_abc" {
		t.Fatalf("expected content hash hash_abc, got %q", got.ContentHash)
	}
	if len(repo.audit) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audit))
	}
	audit := repo.audit[0]
	if audit.Action != domain.AuditSaved {
		t.Fatalf("expected action %q, got %q", domain.AuditSaved, audit.Action)
	}
	detailHash, ok := audit.Details["content_hash"]
	if !ok || detailHash != "hash_abc" {
		t.Fatalf("expected details content_hash=hash_abc, got %v", audit.Details)
	}
	if presigner.DeleteCalled != 0 {
		t.Fatalf("expected DeleteCalled 0, got %d", presigner.DeleteCalled)
	}
}

func TestCommitAutosave_HashMismatch(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{
		ID:       "tpl-1",
		TenantID: "tenant-a",
	}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  2,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/2.docx",
	}
	presigner := &fakePresigner{HeadResult: "hash_actual"}
	svc := application.New(repo, presigner, fakeClock{}, &fakeUUID{})

	_, err := svc.CommitAutosave(context.Background(), application.CommitAutosaveCmd{
		TenantID:            "tenant-a",
		ActorUserID:         "user-a",
		TemplateID:          "tpl-1",
		VersionNumber:       2,
		ExpectedContentHash: "hash_expected",
	})
	if !errors.Is(err, domain.ErrContentHashMismatch) {
		t.Fatalf("expected ErrContentHashMismatch, got %v", err)
	}
	if presigner.DeleteCalled != 1 {
		t.Fatalf("expected DeleteCalled 1, got %d", presigner.DeleteCalled)
	}
}

func TestCommitAutosave_UploadMissing(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{
		ID:       "tpl-1",
		TenantID: "tenant-a",
	}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  4,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/4.docx",
	}
	presigner := &fakePresigner{HeadErr: domain.ErrUploadMissing}
	svc := application.New(repo, presigner, fakeClock{}, &fakeUUID{})

	_, err := svc.CommitAutosave(context.Background(), application.CommitAutosaveCmd{
		TenantID:            "tenant-a",
		ActorUserID:         "user-a",
		TemplateID:          "tpl-1",
		VersionNumber:       4,
		ExpectedContentHash: "hash_abc",
	})
	if !errors.Is(err, domain.ErrUploadMissing) {
		t.Fatalf("expected ErrUploadMissing, got %v", err)
	}
	if presigner.DeleteCalled != 0 {
		t.Fatalf("expected DeleteCalled 0, got %d", presigner.DeleteCalled)
	}
}
