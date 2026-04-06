package application

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestAddVersionCarriesTemplateSnapshotFromCurrentRevision(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedDraftDocument(t, ctx, repo, now)

	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:      doc.ID,
		Number:          1,
		Content:         `<section><p>Initial</p></section>`,
		ContentHash:     contentHash(`<section><p>Initial</p></section>`),
		ChangeSummary:   "Initial",
		ContentSource:   domain.ContentSourceBrowserEditor,
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	version, err := service.AddVersion(ctx, domain.AddVersionCommand{
		DocumentID:    doc.ID,
		Content:       `<section><p>Updated</p></section>`,
		ChangeSummary: "Update",
		TraceID:       "trace-test",
	})
	if err != nil {
		t.Fatalf("AddVersion() error = %v", err)
	}
	if version.TemplateKey != "po-default-canvas" || version.TemplateVersion != 1 {
		t.Fatalf("template snapshot = %q/%d, want po-default-canvas/1", version.TemplateKey, version.TemplateVersion)
	}
}
