package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestMDDMRepository_InsertDraft(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	defer db.Close()

	docID := newTestDocument(t, db)
	repo := NewMDDMRepository(db)

	contentBlocks := json.RawMessage(`{"mddm_version":1,"blocks":[],"template_ref":null}`)

	id, err := repo.InsertDraft(ctx, InsertDraftParams{
		DocumentID:    docID,
		VersionNumber: 1,
		RevisionLabel: "REV01",
		ContentBlocks: contentBlocks,
		ContentHash:   "abcdef",
		CreatedBy:     uuid.New().String(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == uuid.Nil {
		t.Error("expected non-nil id")
	}
}

func TestMDDMRepository_OnlyOneActiveDraftPerDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	ctx := context.Background()
	db := newTestDB(t)
	defer db.Close()

	docID := newTestDocument(t, db)
	repo := NewMDDMRepository(db)

	contentBlocks := json.RawMessage(`{"mddm_version":1,"blocks":[],"template_ref":null}`)

	_, err := repo.InsertDraft(ctx, InsertDraftParams{
		DocumentID:    docID,
		VersionNumber: 1,
		RevisionLabel: "REV01",
		ContentBlocks: contentBlocks,
		ContentHash:   "h1",
		CreatedBy:     "user1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.InsertDraft(ctx, InsertDraftParams{
		DocumentID:    docID,
		VersionNumber: 2,
		RevisionLabel: "REV02",
		ContentBlocks: contentBlocks,
		ContentHash:   "h2",
		CreatedBy:     "user2",
	})
	if err == nil {
		t.Error("expected unique constraint violation on second active draft")
	}
}
