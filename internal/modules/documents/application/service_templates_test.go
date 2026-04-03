package application

import (
	"context"
	"errors"
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

func seedDocumentProfileSchema(t *testing.T, repo *documentmemory.Repository, item domain.DocumentProfileSchemaVersion) {
	t.Helper()

	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), item); err != nil {
		t.Fatalf("upsert schema version %d: %v", item.Version, err)
	}
}
