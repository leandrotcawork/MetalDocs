package application

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func seedCompatiblePOProfileSchemaSet(t *testing.T, repo *documentmemory.Repository) {
	t.Helper()

	schema := map[string]any{
		"sections": []any{
			map[string]any{
				"key": "identificacaoProcesso", "num": "2", "title": "Identificacao do Processo",
				"fields": []any{map[string]any{"key": "objetivo", "label": "Objetivo", "type": "textarea"}},
			},
			map[string]any{
				"key": "visaoGeral", "num": "4", "title": "Visao Geral do Processo",
				"fields": []any{map[string]any{"key": "descricaoProcesso", "label": "Descricao do processo", "type": "rich"}},
			},
		},
	}

	if err := repo.UpsertDocumentTypeDefinition(context.Background(), domain.DocumentTypeDefinition{
		Key: "po", Name: "Procedimento Operacional", ActiveVersion: 1,
		Schema: domain.DocumentTypeSchema{
			Sections: []domain.SectionDef{
				{Key: "identificacaoProcesso", Num: "2", Title: "Identificacao do Processo",
					Fields: []domain.FieldDef{{Key: "objetivo", Label: "Objetivo", Type: "textarea"}}},
				{Key: "visaoGeral", Num: "4", Title: "Visao Geral do Processo",
					Fields: []domain.FieldDef{{Key: "descricaoProcesso", Label: "Descricao do processo", Type: "rich"}}},
			},
		},
	}); err != nil {
		t.Fatalf("seed document type definition: %v", err)
	}

	for _, v := range []int{1, 3} {
		if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
			ProfileCode: "po", Version: v, IsActive: v == 1, ContentSchema: schema,
		}); err != nil {
			t.Fatalf("seed profile schema v%d: %v", v, err)
		}
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func seedDraftDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time) domain.Document {
	t.Helper()

	doc := domain.Document{
		ID:                   "doc-1",
		Title:                "MetalDocs Procedure",
		DocumentType:         "po",
		DocumentProfile:      "po",
		DocumentFamily:       "procedure",
		DocumentSequence:     1,
		DocumentCode:         "PO-001",
		ProfileSchemaVersion: 1,
		OwnerID:              "owner-1",
		BusinessUnit:         "operations",
		Department:           "sgq",
		Classification:       domain.ClassificationInternal,
		Status:               domain.StatusDraft,
		Tags:                 []string{},
		MetadataJSON:         map[string]any{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := repo.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("create document: %v", err)
	}
	return doc
}
