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

	if err := repo.UpsertDocumentTypeDefinition(context.Background(), domain.DocumentTypeDefinition{
		Key: "po", Name: "Procedimento Operacional", ActiveVersion: 1,
	}); err != nil {
		t.Fatalf("seed document type definition: %v", err)
	}

	for _, v := range []int{1, 3} {
		if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
			ProfileCode: "po", Version: v, IsActive: v == 1, ContentSchema: map[string]any{},
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
