package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestResolveDocumentTemplateRejectsIncompatibleSlot(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	seedDocumentProfileSchema(t, repo, domain.DocumentProfileSchemaVersion{
		ProfileCode: "po",
		Version:     1,
		IsActive:    true,
		ContentSchema: map[string]any{
			"sections": []any{
				map[string]any{
					"key":   "visaoGeral",
					"num":   "1",
					"title": "Visao Geral",
					"fields": []any{
						map[string]any{"key": "descricaoProcesso", "label": "Descricao", "type": "rich"},
					},
				},
			},
		},
	})
	seedDocumentProfileSchema(t, repo, domain.DocumentProfileSchemaVersion{
		ProfileCode: "po",
		Version:     2,
		IsActive:    false,
		ContentSchema: map[string]any{
			"sections": []any{
				map[string]any{
					"key":   "visaoGeral",
					"num":   "1",
					"title": "Visao Geral",
					"fields": []any{
						map[string]any{"key": "descricaoProcesso", "label": "Descricao", "type": "textarea"},
					},
				},
			},
		},
	})

	if err := repo.UpsertDocumentTemplateAssignment(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      "doc-1",
		TemplateKey:     "po-governed-canvas",
		TemplateVersion: 1,
		AssignedAt:      time.Unix(0, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert assignment: %v", err)
	}
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-governed-canvas",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 2,
		Name:          "PO governed canvas",
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
		CreatedAt: time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	_, err := service.ResolveDocumentTemplate(ctx, "doc-1", "po")
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestResolveDocumentTemplateRejectsMissingSchemaVersion(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	seedDocumentProfileSchema(t, repo, domain.DocumentProfileSchemaVersion{
		ProfileCode: "po",
		Version:     1,
		IsActive:    true,
		ContentSchema: map[string]any{
			"sections": []any{
				map[string]any{
					"key":   "visaoGeral",
					"num":   "1",
					"title": "Visao Geral",
					"fields": []any{
						map[string]any{"key": "descricaoProcesso", "label": "Descricao", "type": "rich"},
					},
				},
			},
		},
	})

	if err := repo.UpsertDocumentTemplateAssignment(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      "doc-1",
		TemplateKey:     "po-governed-canvas",
		TemplateVersion: 1,
		AssignedAt:      time.Unix(0, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert assignment: %v", err)
	}
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-governed-canvas",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 2,
		Name:          "PO governed canvas",
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
		CreatedAt: time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	_, err := service.ResolveDocumentTemplate(ctx, "doc-1", "po")
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestResolveDocumentTemplateReturnsBrowserTemplateMetadata(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	seedCompatiblePOProfileSchemaSet(t, repo)

	if err := repo.UpsertDocumentTemplateAssignment(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      "doc-1",
		TemplateKey:     "po-browser-template",
		TemplateVersion: 1,
		AssignedAt:      time.Unix(0, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert assignment: %v", err)
	}
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-browser-template",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Browser Template",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><span class="restricted-editing-exception">Objetivo</span></section>`,
		CreatedAt:     time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	got, err := service.ResolveDocumentTemplate(ctx, "doc-1", "po")
	if err != nil {
		t.Fatalf("ResolveDocumentTemplate() error = %v", err)
	}
	if got.Editor != "ckeditor5" || got.ContentFormat != "html" {
		t.Fatalf("template metadata = %#v, want ckeditor5/html", got)
	}
	if !strings.Contains(got.Body, "restricted-editing-exception") {
		t.Fatalf("body = %q, want restricted-editing markup", got.Body)
	}
}

func TestCreateDocumentSeedsBrowserTemplateBody(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})

	seedCompatiblePOProfileSchemaSet(t, repo)

	templateVersion, err := repo.GetDefaultDocumentTemplate(ctx, "po")
	if err != nil {
		t.Fatalf("GetDefaultDocumentTemplate() error = %v", err)
	}

	doc, err := service.CreateDocument(ctx, domain.CreateDocumentCommand{
		DocumentID:      "doc-browser-1",
		Title:           "Browser Seeded Document",
		DocumentType:    "po",
		DocumentProfile: "po",
		OwnerID:         "owner-1",
		BusinessUnit:    "operations",
		Department:      "sgq",
		InitialContent:  `{"legacy":"content"}`,
		TraceID:         "trace-browser-seed",
	})
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}
	if doc.ID != "doc-browser-1" {
		t.Fatalf("document id = %q, want doc-browser-1", doc.ID)
	}

	version, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if version.Content != templateVersion.Body {
		t.Fatalf("version content = %q, want %q", version.Content, templateVersion.Body)
	}
	if version.ContentSource != domain.ContentSourceBrowserEditor {
		t.Fatalf("content source = %q, want %q", version.ContentSource, domain.ContentSourceBrowserEditor)
	}
	if version.TextContent != "Procedimento Operacional Objetivo Preencha o objetivo. Descricao do processo Descreva o processo." {
		t.Fatalf("text content = %q, want sanitized plain text", version.TextContent)
	}
	if version.TemplateKey != "po-default-canvas" || version.TemplateVersion != 1 {
		t.Fatalf("template snapshot = %q/%d, want po-default-canvas/1", version.TemplateKey, version.TemplateVersion)
	}
}

func TestListDocumentTemplatesReturnsProfileCatalog(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-governed-canvas",
		Version:       2,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Governed Canvas v2",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><p>v2</p></section>`,
		CreatedAt:     time.Unix(2, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "wi-default-canvas",
		Version:       1,
		ProfileCode:   "wi",
		SchemaVersion: 1,
		Name:          "WI Default Canvas v1",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><p>wi</p></section>`,
		CreatedAt:     time.Unix(3, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	items, err := service.ListDocumentTemplates(ctx, "po")
	if err != nil {
		t.Fatalf("ListDocumentTemplates() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("template count = %d, want 2", len(items))
	}
	if items[0].TemplateKey != "po-default-canvas" || items[0].Version != 1 {
		t.Fatalf("template[0] = %#v, want po-default-canvas v1", items[0])
	}
	if items[1].TemplateKey != "po-governed-canvas" || items[1].Version != 2 {
		t.Fatalf("template[1] = %#v, want po-governed-canvas v2", items[1])
	}
	for _, item := range items {
		if !item.IsBrowserHTML() {
			t.Fatalf("template metadata = %#v, want browser html", item)
		}
		if item.ProfileCode != "po" {
			t.Fatalf("template profile = %q, want po", item.ProfileCode)
		}
	}
}

func TestAssignDocumentTemplateAuthorizedPersistsDocumentOverride(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})

	seedCompatiblePOProfileSchemaSet(t, repo)
	doc := seedDraftDocument(t, ctx, repo, now)
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-browser-override",
		Version:       2,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Browser Override",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><p>Override</p></section>`,
		CreatedAt:     time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	assignment, err := service.AssignDocumentTemplateAuthorized(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      doc.ID,
		TemplateKey:     "po-browser-override",
		TemplateVersion: 2,
	})
	if err != nil {
		t.Fatalf("AssignDocumentTemplateAuthorized() error = %v", err)
	}
	if assignment.DocumentID != doc.ID {
		t.Fatalf("document id = %q, want %q", assignment.DocumentID, doc.ID)
	}
	if assignment.TemplateKey != "po-browser-override" || assignment.TemplateVersion != 2 {
		t.Fatalf("assignment = %#v, want po-browser-override v2", assignment)
	}
	if !assignment.AssignedAt.Equal(now) {
		t.Fatalf("assigned at = %s, want %s", assignment.AssignedAt, now)
	}

	stored, err := repo.GetDocumentTemplateAssignment(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetDocumentTemplateAssignment() error = %v", err)
	}
	if stored.TemplateKey != "po-browser-override" || stored.TemplateVersion != 2 {
		t.Fatalf("stored assignment = %#v, want po-browser-override v2", stored)
	}
}

func seedDocumentProfileSchema(t *testing.T, repo *documentmemory.Repository, item domain.DocumentProfileSchemaVersion) {
	t.Helper()

	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), item); err != nil {
		t.Fatalf("upsert schema version %d: %v", item.Version, err)
	}
}
