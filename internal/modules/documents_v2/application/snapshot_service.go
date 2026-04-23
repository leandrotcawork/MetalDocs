package application

import (
	"context"

	"metaldocs/internal/modules/documents_v2/domain"
)

// SnapshotTemplateReader loads a template's artifact data for snapshotting.
type SnapshotTemplateReader interface {
	LoadForSnapshot(ctx context.Context, tenantID, templateID string) (domain.TemplateSnapshot, error)
}

// SnapshotWriter persists snapshot columns on a document.
type SnapshotWriter interface {
	WriteSnapshot(ctx context.Context, tenantID, docID string, s domain.TemplateSnapshot) error
}

// SnapshotService copies template artifacts onto a document at creation time.
type SnapshotService struct {
	templates SnapshotTemplateReader
	writer    SnapshotWriter
}

// NewSnapshotService constructs a SnapshotService.
func NewSnapshotService(t SnapshotTemplateReader, w SnapshotWriter) *SnapshotService {
	return &SnapshotService{templates: t, writer: w}
}

// SnapshotFromTemplate loads the template identified by templateID and writes
// all snapshot columns onto the document identified by docID.
func (s *SnapshotService) SnapshotFromTemplate(ctx context.Context, tenantID, docID, templateID string) error {
	snap, err := s.templates.LoadForSnapshot(ctx, tenantID, templateID)
	if err != nil {
		return err
	}
	return s.writer.WriteSnapshot(ctx, tenantID, docID, snap)
}
