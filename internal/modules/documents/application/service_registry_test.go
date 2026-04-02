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
	if len(items) != 1 {
		t.Fatalf("expected 1 schema version, got %d", len(items))
	}

	if _, ok := items[0].ContentSchema["legacy"]; ok {
		t.Fatalf("expected service to replace legacy content schema with type schema")
	}

	sections, ok := items[0].ContentSchema["sections"].([]any)
	if !ok {
		t.Fatalf("expected enriched content schema sections array, got %T", items[0].ContentSchema["sections"])
	}
	if len(sections) == 0 {
		t.Fatalf("expected enriched content schema sections to be non-empty")
	}
}
