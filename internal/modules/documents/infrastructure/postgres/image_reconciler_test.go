package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestImageReconciler_AddsAndRemovesReferences(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	defer db.Close()

	docID := newTestDocument(t, db)
	repo := NewMDDMRepository(db)
	store := NewPostgresByteaStorage(db)
	recon := NewImageReconciler(db)

	// Insert two images with unique content per test run to avoid sha collisions
	uniq := uuid.New().String()
	img1, err := store.Put(ctx, "h1-"+uniq, "image/png", []byte("a-"+uniq))
	if err != nil {
		t.Fatal(err)
	}
	img2, err := store.Put(ctx, "h2-"+uniq, "image/png", []byte("b-"+uniq))
	if err != nil {
		t.Fatal(err)
	}

	// Clean up images after the test
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM metaldocs.document_version_images WHERE image_id IN ($1, $2)`, img1, img2)
		_, _ = db.ExecContext(ctx, `DELETE FROM metaldocs.document_images WHERE id IN ($1, $2)`, img1, img2)
	})

	// Insert draft
	versionID, err := repo.InsertDraft(ctx, InsertDraftParams{
		DocumentID:    docID,
		VersionNumber: 1,
		RevisionLabel: "REV01",
		ContentBlocks: []byte(`{"mddm_version":1,"blocks":[],"template_ref":null}`),
		ContentHash:   "h",
		CreatedBy:     "u",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Reconcile to [img1]
	if err := recon.Reconcile(ctx, versionID, []uuid.UUID{img1}); err != nil {
		t.Fatal(err)
	}

	// Reconcile to [img2] — should remove img1 reference, add img2
	if err := recon.Reconcile(ctx, versionID, []uuid.UUID{img2}); err != nil {
		t.Fatal(err)
	}

	// Verify img1 is no longer referenced by this version
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM metaldocs.document_version_images WHERE document_version_id = $1 AND image_id = $2`, versionID, img1).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("img1 should have been removed, count=%d", count)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM metaldocs.document_version_images WHERE document_version_id = $1 AND image_id = $2`, versionID, img2).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("img2 should have been added, count=%d", count)
	}
}
