package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestGetBrowserEditorBundleReturnsDraftHTML(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}
	if bundle.Body != `<section><p>Original</p></section>` {
		t.Fatalf("bundle body = %q, want original HTML", bundle.Body)
	}
	if bundle.DraftToken == "" {
		t.Fatal("expected draft token")
	}
	if bundle.TemplateSnapshot.TemplateKey != "po-default-canvas" {
		t.Fatalf("template key = %q, want po-default-canvas", bundle.TemplateSnapshot.TemplateKey)
	}
	if !bundle.TemplateSnapshot.IsBrowserHTML() {
		t.Fatalf("template snapshot = %#v, want browser html", bundle.TemplateSnapshot)
	}
}

func TestSaveBrowserContentAuthorizedUpdatesDraftInPlace(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)
	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	version, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if err != nil {
		t.Fatalf("SaveBrowserContentAuthorized() error = %v", err)
	}
	if version.Number != 1 {
		t.Fatalf("version number = %d, want 1", version.Number)
	}
	if version.ContentSource != domain.ContentSourceBrowserEditor {
		t.Fatalf("content source = %q, want %q", version.ContentSource, domain.ContentSourceBrowserEditor)
	}
	if version.Content != `<section><p>Atualizado</p></section>` {
		t.Fatalf("content = %q, want updated HTML", version.Content)
	}

	savedVersion, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() after save error = %v", err)
	}
	if savedVersion.Content != `<section><p>Atualizado</p></section>` {
		t.Fatalf("saved content = %q, want updated HTML", savedVersion.Content)
	}
}

func TestSaveBrowserContentAuthorizedRejectsNonBrowserTemplate(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-governed-docx",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Governed DOCX",
		Editor:        "docx",
		ContentFormat: "json",
		Definition: map[string]any{
			"type": "page",
			"id":   "po-root",
			"children": []any{
				map[string]any{
					"type":  "section-frame",
					"id":    "section-visao-geral",
					"title": "Visao Geral",
					"children": []any{
						map[string]any{"type": "rich-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "rich"},
					},
				},
			},
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertDocumentTemplateVersionForTest() error = %v", err)
	}

	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	current.TemplateKey = "po-governed-docx"
	current.TemplateVersion = 1
	if err := repo.UpdateDraftVersionContentCAS(ctx, current, current.ContentHash); err != nil {
		t.Fatalf("UpdateDraftVersionContentCAS() error = %v", err)
	}

	_, err = service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestGetBrowserEditorBundleRejectsIncompatibleStoredTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

	setStoredBrowserTemplateSnapshotForTest(t, ctx, repo, now, doc.ID, "po-browser-invalid-schema", 99)

	_, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestSaveBrowserContentAuthorizedRejectsIncompatibleStoredTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

	current := setStoredBrowserTemplateSnapshotForTest(t, ctx, repo, now, doc.ID, "po-browser-invalid-schema", 99)

	_, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func seedBrowserDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, body string) domain.Document {
	t.Helper()

	seedCompatiblePOProfileSchemaSet(t, repo)

	doc := seedDraftDocument(t, ctx, repo, now)
	if err := repo.UpsertDocumentTemplateAssignment(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      doc.ID,
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
		AssignedAt:      now,
	}); err != nil {
		t.Fatalf("upsert template assignment: %v", err)
	}
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:      doc.ID,
		Number:          1,
		Content:         body,
		ContentHash:     contentHash(body),
		ChangeSummary:   "Initial browser draft",
		ContentSource:   domain.ContentSourceBrowserEditor,
		TextContent:     plainTextFromHTML(body),
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}
	return doc
}

func setStoredBrowserTemplateSnapshotForTest(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, documentID, templateKey string, schemaVersion int) domain.Version {
	t.Helper()

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   templateKey,
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: schemaVersion,
		Name:          "PO Browser Invalid Snapshot",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><p>Stored invalid browser snapshot</p></section>`,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("UpsertDocumentTemplateVersionForTest() error = %v", err)
	}

	current, err := repo.GetVersion(ctx, documentID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	current.TemplateKey = templateKey
	current.TemplateVersion = 1
	if err := repo.UpdateDraftVersionContentCAS(ctx, current, current.ContentHash); err != nil {
		t.Fatalf("UpdateDraftVersionContentCAS() error = %v", err)
	}
	return current
}
