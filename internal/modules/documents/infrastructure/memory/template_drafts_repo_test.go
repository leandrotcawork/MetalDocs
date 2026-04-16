package memory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents/domain"
)

func newTestDraft(key string) *domain.TemplateDraft {
	return &domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: "po",
		BaseVersion: 0,
		Name:        "Test Draft",
		ThemeJSON:   json.RawMessage(`{"color":"red"}`),
		MetaJSON:    json.RawMessage(`{"author":"alice"}`),
		BlocksJSON:  json.RawMessage(`{"type":"page","children":[]}`),
		CreatedBy:   "alice",
	}
}

func TestTemplateDraftsCAS_HappyPath(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	draft := newTestDraft("tpl-happy")

	// Create (expectedLockVersion = 0)
	saved, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0)
	if err != nil {
		t.Fatalf("first upsert failed: %v", err)
	}
	if saved.LockVersion != 1 {
		t.Fatalf("expected LockVersion 1 after create, got %d", saved.LockVersion)
	}
	if saved.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	// Read back
	got, err := repo.GetTemplateDraft(ctx, "tpl-happy")
	if err != nil {
		t.Fatalf("get after create: %v", err)
	}
	if got.Name != "Test Draft" {
		t.Fatalf("expected name 'Test Draft', got %q", got.Name)
	}

	// Update (expectedLockVersion = 1)
	saved.Name = "Updated Draft"
	saved2, err := repo.UpsertTemplateDraftCAS(ctx, saved, 1)
	if err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}
	if saved2.LockVersion != 2 {
		t.Fatalf("expected LockVersion 2 after update, got %d", saved2.LockVersion)
	}
	if saved2.Name != "Updated Draft" {
		t.Fatalf("expected name 'Updated Draft', got %q", saved2.Name)
	}
}

func TestTemplateDraftsCAS_LockConflict(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	draft := newTestDraft("tpl-conflict")

	// Create first
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Try to update with wrong lock version (0 would create, 2 is wrong for freshly created)
	_, err := repo.UpsertTemplateDraftCAS(ctx, draft, 2)
	if !errors.Is(err, domain.ErrTemplateLockConflict) {
		t.Fatalf("expected ErrTemplateLockConflict, got %v", err)
	}
}

func TestTemplateDraftsCAS_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	draft := newTestDraft("tpl-nonexistent")

	// Attempt update on non-existent draft (expectedLockVersion > 0)
	_, err := repo.UpsertTemplateDraftCAS(ctx, draft, 1)
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Fatalf("expected ErrTemplateDraftNotFound, got %v", err)
	}
}

func TestTemplateDraftDelete(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	draft := newTestDraft("tpl-delete")
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.DeleteTemplateDraft(ctx, "tpl-delete"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := repo.GetTemplateDraft(ctx, "tpl-delete")
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Fatalf("expected ErrTemplateDraftNotFound after delete, got %v", err)
	}
}

func TestTemplateDraftDelete_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	err := repo.DeleteTemplateDraft(ctx, "tpl-missing")
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Fatalf("expected ErrTemplateDraftNotFound, got %v", err)
	}
}

func TestTemplateAuditEvents(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository()

	v := 1
	events := []domain.TemplateAuditEvent{
		{TemplateKey: "tpl-audit", Action: "draft_saved", ActorID: "alice", DiffSummary: "added block"},
		{TemplateKey: "tpl-audit", Version: &v, Action: "published", ActorID: "alice"},
		{TemplateKey: "other-key", Action: "draft_saved", ActorID: "bob"},
	}
	for _, e := range events {
		if err := repo.WriteTemplateAuditEvent(ctx, e); err != nil {
			t.Fatalf("write audit event: %v", err)
		}
	}

	got, err := repo.ListTemplateAuditEvents(ctx, "tpl-audit")
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events for tpl-audit, got %d", len(got))
	}
	if got[0].Action != "draft_saved" {
		t.Fatalf("expected first action 'draft_saved', got %q", got[0].Action)
	}
	if got[1].Version == nil || *got[1].Version != 1 {
		t.Fatalf("expected version 1 on second event")
	}
}

func TestUpdateTemplateVersionStatus_HappyPath(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	// seed a version
	v := domain.DocumentTemplateVersion{
		TemplateKey: "tpl-1",
		Version:     1,
		ProfileCode: "default",
		Status:      "published",
	}
	if err := repo.InsertTemplateVersion(ctx, v); err != nil {
		t.Fatalf("insert template version: %v", err)
	}

	// update status
	if err := repo.UpdateTemplateVersionStatus(ctx, "tpl-1", 1, domain.TemplateStatusDeprecated); err != nil {
		t.Fatalf("update template version status: %v", err)
	}

	// verify
	got, err := repo.GetDocumentTemplateVersion(ctx, "tpl-1", 1)
	if err != nil {
		t.Fatalf("get document template version: %v", err)
	}
	if got.Status != "deprecated" {
		t.Fatalf("expected status 'deprecated', got %q", got.Status)
	}
}

func TestUpdateTemplateVersionStatus_NotFound(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	err := repo.UpdateTemplateVersionStatus(ctx, "nonexistent", 1, domain.TemplateStatusDeprecated)
	if !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Fatalf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestUpdateTemplateDraftStatus_NotFound(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	err := repo.UpdateTemplateDraftStatus(ctx, "missing", domain.TemplateStatusPendingReview)
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Fatalf("expected ErrTemplateDraftNotFound, got %v", err)
	}
}

func TestUpdateTemplateDraftStatus_OK(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	draft := newTestDraft("tpl-status")
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.UpdateTemplateDraftStatus(ctx, "tpl-status", domain.TemplateStatusPendingReview); err != nil {
		t.Fatalf("update draft status: %v", err)
	}

	got, err := repo.GetTemplateDraft(ctx, "tpl-status")
	if err != nil {
		t.Fatalf("get template draft: %v", err)
	}
	if got.DraftStatus != domain.TemplateStatusPendingReview {
		t.Fatalf("expected draft status %q, got %q", domain.TemplateStatusPendingReview, got.DraftStatus)
	}
}

func TestSetTemplateDraftPublished_OK(t *testing.T) {
	repo := NewRepository()
	ctx := context.Background()

	draft := newTestDraft("tpl-published")
	if _, err := repo.UpsertTemplateDraftCAS(ctx, draft, 0); err != nil {
		t.Fatalf("create: %v", err)
	}

	const html = "<html><body>published</body></html>"
	if err := repo.SetTemplateDraftPublished(ctx, "tpl-published", html); err != nil {
		t.Fatalf("set template draft published: %v", err)
	}

	got, err := repo.GetTemplateDraft(ctx, "tpl-published")
	if err != nil {
		t.Fatalf("get template draft: %v", err)
	}
	if got.DraftStatus != domain.TemplateStatusPublished {
		t.Fatalf("expected draft status %q, got %q", domain.TemplateStatusPublished, got.DraftStatus)
	}
	if got.PublishedHTML == nil {
		t.Fatal("expected PublishedHTML to be set")
	}
}
