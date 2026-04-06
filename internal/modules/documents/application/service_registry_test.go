package application

import (
	"context"
	"testing"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestListDocumentProfileSchemasResolvesContentSchemaFromTypeDefinition(t *testing.T) {
	repo := memory.NewRepository()
	service := NewService(repo, nil, nil)

	err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
		ProfileCode:   "po",
		Version:       1,
		IsActive:      true,
		MetadataRules: []domain.MetadataFieldRule{},
		ContentSchema: map[string]any{"legacy": true},
	})
	if err != nil {
		t.Fatalf("upsert schema version: %v", err)
	}

	items, err := service.ListDocumentProfileSchemas(context.Background(), "po")
	if err != nil {
		t.Fatalf("list profile schemas: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 schema version")
	}

	for _, item := range items {
		if _, ok := item.ContentSchema["legacy"]; ok {
			t.Fatalf("expected service to replace legacy content schema with type schema for version %d", item.Version)
		}
		sections, ok := item.ContentSchema["sections"].([]any)
		if !ok {
			t.Fatalf("expected enriched content schema sections array for version %d, got %T", item.Version, item.ContentSchema["sections"])
		}
		if len(sections) == 0 {
			t.Fatalf("expected enriched content schema sections to be non-empty for version %d", item.Version)
		}
	}
}
