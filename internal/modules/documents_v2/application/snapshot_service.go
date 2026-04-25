package application

import (
	"context"
	"encoding/json"
	"fmt"

	"metaldocs/internal/modules/documents_v2/domain"
	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

// SnapshotTemplateReader loads a template's artifact data for snapshotting.
type SnapshotTemplateReader interface {
	LoadForSnapshot(ctx context.Context, tenantID, templateID string) (domain.TemplateSnapshot, error)
}

// SnapshotWriter persists snapshot columns on a document.
type SnapshotWriter interface {
	WriteSnapshot(ctx context.Context, tenantID, docID string, s domain.TemplateSnapshot) error
}

// PlaceholderValueSeeder seeds default placeholder value rows for a revision.
type PlaceholderValueSeeder interface {
	SeedDefaults(ctx context.Context, tenantID, revisionID string, phs []templatesdomain.Placeholder) error
}

// SnapshotService copies template artifacts onto a document at creation time
// and optionally seeds default placeholder value rows.
type SnapshotService struct {
	templates SnapshotTemplateReader
	writer    SnapshotWriter
	seeder    PlaceholderValueSeeder // optional
}

// NewSnapshotService constructs a SnapshotService without a seeder.
func NewSnapshotService(t SnapshotTemplateReader, w SnapshotWriter) *SnapshotService {
	return &SnapshotService{templates: t, writer: w}
}

// NewSnapshotServiceWithSeeder constructs a SnapshotService with a seeder
// that inserts default placeholder value rows after the snapshot is written.
func NewSnapshotServiceWithSeeder(t SnapshotTemplateReader, w SnapshotWriter, s PlaceholderValueSeeder) *SnapshotService {
	return &SnapshotService{templates: t, writer: w, seeder: s}
}

// SnapshotFromTemplate loads the template identified by templateID, writes all
// snapshot columns onto the document identified by docID, and seeds default
// placeholder value rows for revisionID (if a seeder is configured).
func (s *SnapshotService) SnapshotFromTemplate(ctx context.Context, tenantID, docID, revisionID, templateID string) error {
	snap, err := s.templates.LoadForSnapshot(ctx, tenantID, templateID)
	if err != nil {
		return err
	}
	if err := s.writer.WriteSnapshot(ctx, tenantID, docID, snap); err != nil {
		return err
	}
	if s.seeder == nil {
		return nil
	}
	// Parse required placeholders from the snapshot JSON.
	phs, err := parseRequiredPlaceholders(snap.PlaceholderSchemaJSON)
	if err != nil {
		return fmt.Errorf("parse placeholder schema: %w", err)
	}
	if len(phs) == 0 {
		return nil
	}
	return s.seeder.SeedDefaults(ctx, tenantID, revisionID, phs)
}

// parseRequiredPlaceholders extracts placeholders with Required=true from
// the placeholder schema JSON blob. Returns empty slice on empty/nil input.
func parseRequiredPlaceholders(schemaJSON []byte) ([]templatesdomain.Placeholder, error) {
	if len(schemaJSON) == 0 {
		return nil, nil
	}
	var schema struct {
		Placeholders []templatesdomain.Placeholder `json:"placeholders"`
	}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, err
	}
	var out []templatesdomain.Placeholder
	for _, p := range schema.Placeholders {
		if p.Required {
			out = append(out, p)
		}
	}
	return out, nil
}
