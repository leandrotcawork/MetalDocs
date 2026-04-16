package application_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func makeCK5TemplateSvc(t *testing.T) (*application.Service, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	return svc, repo
}

func seedDraft(t *testing.T, repo *memory.Repository, key string) {
	t.Helper()
	emptyBlocks, _ := json.Marshal(map[string]any{"type": "doc", "content": []any{}})
	draft := &domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: "po",
		Name:        "Test Draft",
		BlocksJSON:  emptyBlocks,
		LockVersion: 1,
		CreatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repo.UpsertTemplateDraftForTest(draft); err != nil {
		t.Fatalf("seed draft: %v", err)
	}
}

func TestGetCK5TemplateDraftContent_EmptyWhenNoCK5Key(t *testing.T) {
	svc, repo := makeCK5TemplateSvc(t)
	seedDraft(t, repo, "tpl-1")

	html, manifest, err := svc.GetCK5TemplateDraftContent(context.Background(), "tpl-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "" {
		t.Errorf("expected empty html, got %q", html)
	}
	if manifest == nil {
		t.Error("manifest should be non-nil (empty default)")
	}
}

func TestGetCK5TemplateDraftContent_NotFound(t *testing.T) {
	svc, _ := makeCK5TemplateSvc(t)
	_, _, err := svc.GetCK5TemplateDraftContent(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing draft")
	}
}

func TestSaveAndGetCK5TemplateDraftContent_RoundTrip(t *testing.T) {
	svc, repo := makeCK5TemplateSvc(t)
	seedDraft(t, repo, "tpl-2")

	ctx := context.Background()
	manifest := map[string]any{"fields": []any{}}
	if err := svc.SaveCK5TemplateDraftAuthorized(ctx, "tpl-2", "<p>CK5 tpl</p>", manifest, "user-1"); err != nil {
		t.Fatalf("save: %v", err)
	}

	html, got, err := svc.GetCK5TemplateDraftContent(ctx, "tpl-2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if html != "<p>CK5 tpl</p>" {
		t.Errorf("html: got %q, want <p>CK5 tpl</p>", html)
	}
	if got == nil {
		t.Error("manifest is nil")
	}
}

func TestSaveCK5TemplateDraftAuthorized_NotFound(t *testing.T) {
	svc, _ := makeCK5TemplateSvc(t)
	err := svc.SaveCK5TemplateDraftAuthorized(context.Background(), "missing", "<p>x</p>", nil, "u")
	if err == nil {
		t.Fatal("expected error for missing draft")
	}
}

// TestSaveCK5TemplateDraftAuthorized_PreservesNonCK5Keys verifies merge semantics:
// pre-existing non-_ck5 keys in BlocksJSON survive a CK5 save unchanged.
func TestSaveCK5TemplateDraftAuthorized_PreservesNonCK5Keys(t *testing.T) {
	svc, repo := makeCK5TemplateSvc(t)

	// Seed a draft whose BlocksJSON contains a "type" key (BlockNote-style).
	blocknotePayload, _ := json.Marshal(map[string]any{
		"type":    "doc",
		"content": []any{},
	})
	_ = repo.UpsertTemplateDraftForTest(&domain.TemplateDraft{
		TemplateKey: "tpl-preserve",
		ProfileCode: "po",
		Name:        "Preserve Test",
		BlocksJSON:  blocknotePayload,
		LockVersion: 1,
		CreatedBy:   "test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	ctx := context.Background()
	if err := svc.SaveCK5TemplateDraftAuthorized(ctx, "tpl-preserve", "<p>ck5</p>", nil, "u"); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Read back raw BlocksJSON and verify both "type" (BlockNote) and "_ck5" keys coexist.
	draft, err := repo.GetTemplateDraft(ctx, "tpl-preserve")
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	var merged map[string]any
	if err := json.Unmarshal(draft.BlocksJSON, &merged); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if merged["type"] != "doc" {
		t.Errorf("pre-existing 'type' key clobbered; got %v", merged["type"])
	}
	if merged["_ck5"] == nil {
		t.Error("_ck5 key missing after save")
	}
}
