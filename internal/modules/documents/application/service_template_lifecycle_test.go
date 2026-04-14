package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

// ctxWithRole returns a context carrying the given IAM role, triggering RBAC checks.
func ctxWithRole(role iamdomain.Role) context.Context {
	return iamdomain.WithAuthContext(context.Background(), "user-test", []iamdomain.Role{role})
}

// ctxBypassed returns a context with no user/roles — RBAC is bypassed.
func ctxBypassed() context.Context {
	return context.Background()
}

// --- CreateDraftAuthorized --------------------------------------------------

func TestCreateDraftAuthorized_HappyPath(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "My Draft", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}
	if draft == nil {
		t.Fatal("expected non-nil draft")
	}
	if draft.ProfileCode != "po" {
		t.Errorf("ProfileCode = %q, want po", draft.ProfileCode)
	}
	if draft.Name != "My Draft" {
		t.Errorf("Name = %q, want My Draft", draft.Name)
	}
	if draft.TemplateKey == "" {
		t.Error("expected non-empty TemplateKey")
	}
	if draft.LockVersion != 1 {
		t.Errorf("LockVersion = %d, want 1", draft.LockVersion)
	}

	// Verify audit event was written.
	events, err := repo.ListTemplateAuditEvents(ctxBypassed(), draft.TemplateKey)
	if err != nil {
		t.Fatalf("ListTemplateAuditEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(events))
	}
	if events[0].Action != "draft_created" {
		t.Errorf("audit action = %q, want draft_created", events[0].Action)
	}
	if events[0].ActorID != "actor-1" {
		t.Errorf("audit actorID = %q, want actor-1", events[0].ActorID)
	}
}

func TestCreateDraftAuthorized_RBACDenied(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	// Viewer role has no template.edit capability.
	_, err := svc.CreateDraftAuthorized(ctxWithRole(iamdomain.RoleViewer), "po", "Draft", "actor-1")
	if err == nil {
		t.Fatal("expected RBAC denial, got nil error")
	}
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound (RBAC mask)", err)
	}
}

// --- SaveDraftAuthorized ----------------------------------------------------

func TestSaveDraftAuthorized_HappyPath(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "Draft to Save", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	blocks := json.RawMessage(`{"blocks":[]}`)
	theme := json.RawMessage(`{"color":"blue"}`)
	meta := json.RawMessage(`{"description":"test"}`)

	saved, err := svc.SaveDraftAuthorized(ctxBypassed(), draft.TemplateKey, blocks, theme, meta, draft.LockVersion, "actor-1")
	if err != nil {
		t.Fatalf("SaveDraftAuthorized() error = %v", err)
	}
	if saved.LockVersion <= draft.LockVersion {
		t.Errorf("LockVersion after save = %d, want > %d", saved.LockVersion, draft.LockVersion)
	}

	// Audit: create + save = 2 events.
	events, err := repo.ListTemplateAuditEvents(ctxBypassed(), draft.TemplateKey)
	if err != nil {
		t.Fatalf("ListTemplateAuditEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("audit event count = %d, want 2", len(events))
	}
	if events[1].Action != "draft_saved" {
		t.Errorf("audit action = %q, want draft_saved", events[1].Action)
	}
}

func TestSaveDraftAuthorized_CASConflict(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "Draft", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	blocks := json.RawMessage(`{"blocks":[]}`)
	// Advance lockVersion by saving once successfully.
	_, err = svc.SaveDraftAuthorized(ctxBypassed(), draft.TemplateKey, blocks, nil, nil, draft.LockVersion, "actor-1")
	if err != nil {
		t.Fatalf("first save error = %v", err)
	}

	// Now use the original (stale) lock version — should conflict.
	_, err = svc.SaveDraftAuthorized(ctxBypassed(), draft.TemplateKey, blocks, nil, nil, draft.LockVersion, "actor-1")
	if !errors.Is(err, domain.ErrTemplateLockConflict) {
		t.Errorf("err = %v, want ErrTemplateLockConflict", err)
	}
}

func TestSaveDraftAuthorized_NotFound(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	_, err := svc.SaveDraftAuthorized(ctxBypassed(), "nonexistent-key", nil, nil, nil, 1, "actor-1")
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Errorf("err = %v, want ErrTemplateDraftNotFound", err)
	}
}

func TestSaveDraftAuthorized_RBACDenied(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	_, err := svc.SaveDraftAuthorized(ctxWithRole(iamdomain.RoleViewer), "any-key", nil, nil, nil, 1, "actor-1")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound (RBAC mask)", err)
	}
}

// --- PublishAuthorized ------------------------------------------------------

func TestPublishAuthorized_HappyPath(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "Ready to Publish", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	tv, err := svc.PublishAuthorized(ctxBypassed(), draft.TemplateKey, draft.LockVersion, "actor-1")
	if err != nil {
		t.Fatalf("PublishAuthorized() error = %v", err)
	}
	if tv == nil {
		t.Fatal("expected non-nil DocumentTemplateVersion")
	}
	if tv.Version != 1 {
		t.Errorf("version = %d, want 1 (BaseVersion 0 + 1)", tv.Version)
	}
	if tv.Status != string(domain.TemplateStatusPublished) {
		t.Errorf("status = %q, want published", tv.Status)
	}

	// Draft should be deleted after publish.
	_, err = repo.GetTemplateDraft(ctxBypassed(), draft.TemplateKey)
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Errorf("draft should be deleted after publish, got err = %v", err)
	}

	// Audit: create + published.
	events, err := repo.ListTemplateAuditEvents(ctxBypassed(), draft.TemplateKey)
	if err != nil {
		t.Fatalf("ListTemplateAuditEvents() error = %v", err)
	}
	foundPublished := false
	for _, e := range events {
		if e.Action == "published" {
			foundPublished = true
			if e.Version == nil || *e.Version != 1 {
				t.Errorf("publish audit version = %v, want &1", e.Version)
			}
		}
	}
	if !foundPublished {
		t.Error("expected published audit event, not found")
	}
}

func TestPublishAuthorized_BlockedByStrippedFields(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "Has Stripped", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	// Manually inject HasStrippedFields via a direct repo upsert.
	stripped := &domain.TemplateDraft{
		TemplateKey:       draft.TemplateKey,
		ProfileCode:       draft.ProfileCode,
		Name:              draft.Name,
		HasStrippedFields: true,
	}
	_, err = repo.UpsertTemplateDraftCAS(ctxBypassed(), stripped, draft.LockVersion)
	if err != nil {
		t.Fatalf("inject stripped fields: %v", err)
	}

	// Reload to get updated LockVersion.
	updated, err := repo.GetTemplateDraft(ctxBypassed(), draft.TemplateKey)
	if err != nil {
		t.Fatalf("GetTemplateDraft() error = %v", err)
	}

	_, err = svc.PublishAuthorized(ctxBypassed(), draft.TemplateKey, updated.LockVersion, "actor-1")
	if !errors.Is(err, domain.ErrTemplateHasStrippedFields) {
		t.Errorf("err = %v, want ErrTemplateHasStrippedFields", err)
	}
}

func TestPublishAuthorized_DraftNotFound(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	_, err := svc.PublishAuthorized(ctxBypassed(), "no-such-draft", 1, "actor-1")
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Errorf("err = %v, want ErrTemplateDraftNotFound", err)
	}
}

func TestPublishAuthorized_RBACDenied(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	// Editor role has no template.publish capability.
	_, err := svc.PublishAuthorized(ctxWithRole(iamdomain.RoleEditor), "any-key", 1, "actor-1")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound (RBAC mask)", err)
	}
}

func TestPublishAuthorized_LockConflict(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "Lock Test", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	// Try to publish with a stale lock version.
	_, err = svc.PublishAuthorized(ctxBypassed(), draft.TemplateKey, draft.LockVersion+1, "actor-1")
	if !errors.Is(err, domain.ErrTemplateLockConflict) {
		t.Errorf("err = %v, want ErrTemplateLockConflict", err)
	}
}

func TestPublishAuthorized_StrictValidation(t *testing.T) {
	t.Skip("phase 4 — strict codec not yet wired")
}

// --- DiscardDraftAuthorized -------------------------------------------------

func TestDiscardDraftAuthorized_HappyPath(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "To Discard", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	if err := svc.DiscardDraftAuthorized(ctxBypassed(), draft.TemplateKey, "actor-1"); err != nil {
		t.Fatalf("DiscardDraftAuthorized() error = %v", err)
	}

	// Draft should be gone.
	_, err = repo.GetTemplateDraft(ctxBypassed(), draft.TemplateKey)
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Errorf("draft should be deleted after discard, got err = %v", err)
	}

	// Audit: create + draft_discarded.
	events, err := repo.ListTemplateAuditEvents(ctxBypassed(), draft.TemplateKey)
	if err != nil {
		t.Fatalf("ListTemplateAuditEvents() error = %v", err)
	}
	foundDiscard := false
	for _, e := range events {
		if e.Action == "draft_discarded" {
			foundDiscard = true
		}
	}
	if !foundDiscard {
		t.Error("expected draft_discarded audit event, not found")
	}
}

func TestDiscardDraftAuthorized_NotFound(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	err := svc.DiscardDraftAuthorized(ctxBypassed(), "no-draft", "actor-1")
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Errorf("err = %v, want ErrTemplateDraftNotFound", err)
	}
}

func TestDiscardDraftAuthorized_RBACDenied(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	err := svc.DiscardDraftAuthorized(ctxWithRole(iamdomain.RoleViewer), "any-key", "actor-1")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound (RBAC mask)", err)
	}
}

// --- DeprecateAuthorized ----------------------------------------------------

func TestDeprecateAuthorized_HappyPath(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	// Publish a version first.
	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "To Deprecate", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}
	tv, err := svc.PublishAuthorized(ctxBypassed(), draft.TemplateKey, draft.LockVersion, "actor-1")
	if err != nil {
		t.Fatalf("PublishAuthorized() error = %v", err)
	}

	if err := svc.DeprecateAuthorized(ctxBypassed(), tv.TemplateKey, tv.Version, "actor-1"); err != nil {
		t.Fatalf("DeprecateAuthorized() error = %v", err)
	}

	// Audit should have "deprecated" entry.
	events, err := repo.ListTemplateAuditEvents(ctxBypassed(), tv.TemplateKey)
	if err != nil {
		t.Fatalf("ListTemplateAuditEvents() error = %v", err)
	}
	foundDeprecated := false
	for _, e := range events {
		if e.Action == "deprecated" {
			foundDeprecated = true
			if e.Version == nil || *e.Version != tv.Version {
				t.Errorf("deprecated audit version = %v, want &%d", e.Version, tv.Version)
			}
		}
	}
	if !foundDeprecated {
		t.Error("expected deprecated audit event, not found")
	}
}

func TestDeprecateAuthorized_VersionNotFound(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	err := svc.DeprecateAuthorized(ctxBypassed(), "no-such-key", 1, "actor-1")
	if !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Errorf("err = %v, want ErrTemplateNotFound", err)
	}
}

func TestDeprecateAuthorized_RBACDenied(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	// Editor role has no template.publish capability.
	err := svc.DeprecateAuthorized(ctxWithRole(iamdomain.RoleEditor), "any-key", 1, "actor-1")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Errorf("err = %v, want ErrDocumentNotFound (RBAC mask)", err)
	}
}

// --- Read-only methods ------------------------------------------------------

func TestGetTemplateDraft_NotFound(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	_, err := svc.GetTemplateDraft(ctxBypassed(), "no-draft")
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		t.Errorf("err = %v, want ErrTemplateDraftNotFound", err)
	}
}

func TestGetTemplateDraft_Found(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	draft, err := svc.CreateDraftAuthorized(ctxBypassed(), "po", "Readable", "actor-1")
	if err != nil {
		t.Fatalf("CreateDraftAuthorized() error = %v", err)
	}

	got, err := svc.GetTemplateDraft(ctxBypassed(), draft.TemplateKey)
	if err != nil {
		t.Fatalf("GetTemplateDraft() error = %v", err)
	}
	if got.TemplateKey != draft.TemplateKey {
		t.Errorf("TemplateKey = %q, want %q", got.TemplateKey, draft.TemplateKey)
	}
}

func TestListTemplateAuditEvents_ReturnsEmpty(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := NewService(repo, nil, nil)

	events, err := svc.ListTemplateAuditEvents(ctxBypassed(), "no-events-key")
	if err != nil {
		t.Fatalf("ListTemplateAuditEvents() error = %v", err)
	}
	if len(events) != 0 {
		t.Errorf("event count = %d, want 0", len(events))
	}
}
