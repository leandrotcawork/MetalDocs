package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type fakeExportRepo struct {
	version *exportVersion
	err     error
}

func (f *fakeExportRepo) GetVersion(ctx context.Context, versionID uuid.UUID) (*exportVersion, error) {
	return f.version, f.err
}

type fakeRendererForExport struct{}

func (r *fakeRendererForExport) RenderDocx(ctx context.Context, content []byte) ([]byte, error) {
	return []byte("fresh-render"), nil
}

func TestExportService_ReleasedServesCachedBytes(t *testing.T) {
	svc := NewExportService(&fakeExportRepo{
		version: &exportVersion{
			Status:    "released",
			DocxBytes: []byte("cached-docx"),
		},
	}, &fakeRendererForExport{})

	got, err := svc.ExportDocx(context.Background(), uuid.New(), "debug")
	if err != nil {
		t.Fatalf("ExportDocx() error = %v", err)
	}

	if string(got) != "cached-docx" {
		t.Fatalf("ExportDocx() = %q, want %q", got, "cached-docx")
	}
}

func TestExportService_ArchivedServesCachedBytes(t *testing.T) {
	svc := NewExportService(&fakeExportRepo{
		version: &exportVersion{
			Status:    "archived",
			DocxBytes: []byte("archived-docx"),
		},
	}, &fakeRendererForExport{})

	got, err := svc.ExportDocx(context.Background(), uuid.New(), "production")
	if err != nil {
		t.Fatalf("ExportDocx() error = %v", err)
	}

	if string(got) != "archived-docx" {
		t.Fatalf("ExportDocx() = %q, want %q", got, "archived-docx")
	}
}

func TestExportService_DraftRendersFresh(t *testing.T) {
	svc := NewExportService(&fakeExportRepo{
		version: &exportVersion{
			Status:        "draft",
			ContentBlocks: []byte("draft-content"),
			DocxBytes:     []byte("stale-docx"),
		},
	}, &fakeRendererForExport{})

	got, err := svc.ExportDocx(context.Background(), uuid.New(), "debug")
	if err != nil {
		t.Fatalf("ExportDocx() error = %v", err)
	}

	if string(got) != "fresh-render" {
		t.Fatalf("ExportDocx() = %q, want %q", got, "fresh-render")
	}
}
