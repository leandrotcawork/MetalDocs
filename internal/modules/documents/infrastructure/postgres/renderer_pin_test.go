package postgres

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

// newTestDocumentWithVersion creates a document + version 1 and returns the document ID.
// It reuses newTestDocument for the document row and calls repo.SaveVersion for the version.
func newTestDocumentWithVersion(t *testing.T, ctx context.Context, repo *Repository, docID string) {
	t.Helper()
	err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    docID,
		Number:        1,
		Content:       "{}",
		ContentHash:   "initial-hash",
		ChangeSummary: "initial version",
		ContentSource: domain.ContentSourceNative,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("newTestDocumentWithVersion SaveVersion: %v", err)
	}
}

func TestRepository_SetVersionRendererPin_Roundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	db := newTestDB(t)
	defer db.Close()
	repo := NewRepository(db)

	ctx := context.Background()
	docID := newTestDocument(t, db)
	newTestDocumentWithVersion(t, ctx, repo, docID)

	pin := &domain.RendererPin{
		RendererVersion: "1.0.0",
		LayoutIRHash:    "abcdef0123456789",
		TemplateKey:     "po-mddm-canvas",
		TemplateVersion: 1,
		PinnedAt:        time.Now().UTC().Truncate(time.Second),
	}

	if err := repo.SetVersionRendererPin(ctx, docID, 1, pin); err != nil {
		t.Fatalf("SetVersionRendererPin: %v", err)
	}

	got, err := repo.GetVersion(ctx, docID, 1)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got.RendererPin == nil {
		t.Fatalf("expected RendererPin to be populated")
	}
	if got.RendererPin.RendererVersion != "1.0.0" || got.RendererPin.LayoutIRHash != "abcdef0123456789" {
		t.Fatalf("roundtrip mismatch: %+v", got.RendererPin)
	}

	// Clearing the pin must set the column back to NULL.
	if err := repo.SetVersionRendererPin(ctx, docID, 1, nil); err != nil {
		t.Fatalf("SetVersionRendererPin nil: %v", err)
	}
	got, err = repo.GetVersion(ctx, docID, 1)
	if err != nil {
		t.Fatalf("GetVersion after clear: %v", err)
	}
	if got.RendererPin != nil {
		t.Fatalf("expected RendererPin to be cleared, got %+v", got.RendererPin)
	}
}

func TestRepository_SetVersionRendererPin_FailsWhenVersionMissing(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	db := newTestDB(t)
	defer db.Close()
	repo := NewRepository(db)

	err := repo.SetVersionRendererPin(context.Background(), "nonexistent-doc", 1, &domain.RendererPin{
		RendererVersion: "1.0.0",
		LayoutIRHash:    "h",
		TemplateKey:     "k",
		TemplateVersion: 1,
	})
	if err == nil {
		t.Fatalf("expected error when version row does not exist")
	}
}
