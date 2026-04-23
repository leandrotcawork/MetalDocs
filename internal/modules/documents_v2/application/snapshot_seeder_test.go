package application_test

// snapshot_seeder_test.go — unit test asserting PlaceholderValueSeeder is called
// when SnapshotService has a seeder wired and PlaceholderSchemaJSON has required placeholders.

import (
	"context"
	"encoding/json"
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

// seedRecorder records SeedDefaults calls.
type seedRecorder struct {
	called bool
	phs    []templatesdomain.Placeholder
}

func (r *seedRecorder) SeedDefaults(_ context.Context, _, _ string, phs []templatesdomain.Placeholder) error {
	r.called = true
	r.phs = phs
	return nil
}

// noopSnapshotWriter discards writes.
type noopSnapshotWriter struct{}

func (noopSnapshotWriter) WriteSnapshot(_ context.Context, _, _ string, _ domain.TemplateSnapshot) error {
	return nil
}

// seedTemplateReader returns a snapshot with two required placeholders.
type seedTemplateReader struct{}

func (seedTemplateReader) LoadForSnapshot(_ context.Context, _, _ string) (domain.TemplateSnapshot, error) {
	schema := struct {
		Placeholders []templatesdomain.Placeholder `json:"placeholders"`
	}{
		Placeholders: []templatesdomain.Placeholder{
			{ID: "ph1", Required: true, Type: templatesdomain.PHText},
			{ID: "ph2", Required: false, Type: templatesdomain.PHText},
		},
	}
	b, _ := json.Marshal(schema)
	return domain.TemplateSnapshot{
		PlaceholderSchemaJSON: b,
		CompositionJSON:       []byte(`{}`),
		ZonesSchemaJSON:       []byte(`{}`),
	}, nil
}

func TestSnapshotService_SeedsRequiredPlaceholders(t *testing.T) {
	seeder := &seedRecorder{}
	svc := application.NewSnapshotServiceWithSeeder(seedTemplateReader{}, noopSnapshotWriter{}, seeder)

	if err := svc.SnapshotFromTemplate(context.Background(), "tenant-1", "doc-1", "rev-1", "tmpl-1"); err != nil {
		t.Fatalf("SnapshotFromTemplate: %v", err)
	}
	if !seeder.called {
		t.Fatal("expected seeder to be called, was not")
	}
	if len(seeder.phs) != 1 || seeder.phs[0].ID != "ph1" {
		t.Fatalf("expected seeder called with [ph1], got %+v", seeder.phs)
	}
}
