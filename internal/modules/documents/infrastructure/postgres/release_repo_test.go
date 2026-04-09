package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestReleaseRepo_SingleTransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}

	ctx := context.Background()
	db := newTestDB(t)
	defer db.Close()

	docID := newTestDocument(t, db)
	mddmRepo := NewMDDMRepository(db)
	releaseRepo := NewReleaseRepo(db)

	releasedContent := json.RawMessage(`{"mddm_version":1,"blocks":[{"id":"old"}],"template_ref":null}`)
	releasedTemplateRef := json.RawMessage(`{"template_id":"tpl-old","version":1}`)

	var releasedID uuid.UUID
	if err := db.QueryRowContext(ctx, `
		INSERT INTO metaldocs.document_versions_mddm (
			document_id, version_number, revision_label, status,
			content_blocks, docx_bytes, template_ref, content_hash, created_by, approved_at, approved_by
		)
		VALUES ($1, $2, $3, 'released', $4, $5, $6, $7, $8, now(), $9)
		RETURNING id
	`, docID, 1, "REV00", releasedContent, []byte("released-docx"), releasedTemplateRef, "released-hash", "creator-1", "approver-0").Scan(&releasedID); err != nil {
		t.Fatalf("insert released version: %v", err)
	}

	draftID, err := mddmRepo.InsertDraft(ctx, InsertDraftParams{
		DocumentID:    docID,
		VersionNumber: 2,
		RevisionLabel: "REV01",
		ContentBlocks: json.RawMessage(`{"mddm_version":1,"blocks":[{"id":"draft"}],"template_ref":null}`),
		ContentHash:   "draft-hash",
		TemplateRef:   json.RawMessage(`{"template_id":"tpl-draft","version":2}`),
		CreatedBy:     "creator-2",
	})
	if err != nil {
		t.Fatalf("insert draft: %v", err)
	}

	prevVersionID, prevContentBlocks, prevDocx, err := releaseRepo.ArchivePreviousReleased(ctx, docID)
	if err != nil {
		t.Fatalf("archive previous released: %v", err)
	}
	if prevVersionID != releasedID {
		t.Fatalf("archive returned wrong version id: got %s want %s", prevVersionID, releasedID)
	}
	if !jsonEqual(prevContentBlocks, releasedContent) {
		t.Fatalf("archive returned wrong content blocks: got %s want %s", prevContentBlocks, releasedContent)
	}
	if !bytes.Equal(prevDocx, []byte("released-docx")) {
		t.Fatalf("archive returned wrong docx bytes: %q", prevDocx)
	}

	if err := releaseRepo.PromoteDraftToReleased(ctx, draftID, []byte("draft-docx"), "approver-1"); err != nil {
		t.Fatalf("promote draft: %v", err)
	}

	if err := releaseRepo.StoreRevisionDiff(ctx, draftID, json.RawMessage(`{`)); err == nil {
		t.Fatal("expected invalid revision diff to fail")
	}

	releasedState := mustLoadReleaseVersionState(t, ctx, db, releasedID)
	if releasedState.status != "released" {
		t.Fatalf("released version status after rollback = %s, want released", releasedState.status)
	}
	if !jsonEqual(releasedState.contentBlocks, releasedContent) {
		t.Fatalf("released version content_blocks changed after rollback: got %s want %s", releasedState.contentBlocks, releasedContent)
	}
	if !bytes.Equal(releasedState.docxBytes, []byte("released-docx")) {
		t.Fatalf("released version docx changed after rollback: got %q", releasedState.docxBytes)
	}

	draftState := mustLoadReleaseVersionState(t, ctx, db, draftID)
	if draftState.status != "draft" {
		t.Fatalf("draft status after rollback = %s, want draft", draftState.status)
	}
	if draftState.approvedBy.Valid {
		t.Fatalf("draft approved_by should be null after rollback, got %s", draftState.approvedBy.String)
	}
	if len(draftState.docxBytes) != 0 {
		t.Fatalf("draft docx_bytes should be null after rollback, got %q", draftState.docxBytes)
	}
	if draftState.revisionDiff != nil {
		t.Fatalf("draft revision_diff should be null after rollback, got %s", draftState.revisionDiff)
	}

	prevVersionID, prevContentBlocks, _, err = releaseRepo.ArchivePreviousReleased(ctx, docID)
	if err != nil {
		t.Fatalf("archive previous released after rollback: %v", err)
	}
	if prevVersionID != releasedID {
		t.Fatalf("archive after rollback returned wrong version id: got %s want %s", prevVersionID, releasedID)
	}
	if !jsonEqual(prevContentBlocks, releasedContent) {
		t.Fatalf("archive after rollback returned wrong content blocks: got %s want %s", prevContentBlocks, releasedContent)
	}

	if err := releaseRepo.PromoteDraftToReleased(ctx, draftID, []byte("draft-docx"), "approver-1"); err != nil {
		t.Fatalf("promote draft after rollback: %v", err)
	}
	if err := releaseRepo.StoreRevisionDiff(ctx, draftID, json.RawMessage(`{"added":[],"removed":[],"modified":[]}`)); err != nil {
		t.Fatalf("store revision diff after rollback: %v", err)
	}
	if err := releaseRepo.DeleteImageRefs(ctx, releasedID); err != nil {
		t.Fatalf("delete image refs: %v", err)
	}
	if err := releaseRepo.CleanupOrphanImages(ctx); err != nil {
		t.Fatalf("cleanup orphan images: %v", err)
	}

	releasedState = mustLoadReleaseVersionState(t, ctx, db, releasedID)
	if releasedState.status != "archived" {
		t.Fatalf("released version status after commit = %s, want archived", releasedState.status)
	}
	if releasedState.contentBlocks != nil {
		t.Fatalf("archived version content_blocks should be null, got %s", releasedState.contentBlocks)
	}

	draftState = mustLoadReleaseVersionState(t, ctx, db, draftID)
	if draftState.status != "released" {
		t.Fatalf("draft status after commit = %s, want released", draftState.status)
	}
	if !draftState.approvedBy.Valid || draftState.approvedBy.String != "approver-1" {
		t.Fatalf("draft approved_by after commit = %+v, want approver-1", draftState.approvedBy)
	}
	if !bytes.Equal(draftState.docxBytes, []byte("draft-docx")) {
		t.Fatalf("draft docx_bytes after commit = %q, want draft-docx", draftState.docxBytes)
	}
	if !jsonEqual(draftState.revisionDiff, []byte(`{"added":[],"removed":[],"modified":[]}`)) {
		t.Fatalf("draft revision_diff after commit = %s", draftState.revisionDiff)
	}
}

func TestReleaseRepo_ContextsKeepIndependentTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}

	rootCtx := context.Background()
	db := newTestDB(t)
	defer db.Close()

	docAID, releasedAID, draftAID := seedReleaseRepoDocument(t, rootCtx, db, "A")
	docBID, releasedBID, draftBID := seedReleaseRepoDocument(t, rootCtx, db, "B")

	releaseRepo := NewReleaseRepo(db)
	ctxA := context.WithValue(rootCtx, releaseRepoTestContextKey("tx"), "A")
	ctxB := context.WithValue(rootCtx, releaseRepoTestContextKey("tx"), "B")

	if _, _, _, err := releaseRepo.ArchivePreviousReleased(ctxA, docAID); err != nil {
		t.Fatalf("archive previous released A: %v", err)
	}
	if err := releaseRepo.PromoteDraftToReleased(ctxA, draftAID, []byte("draft-docx-A"), "approver-A"); err != nil {
		t.Fatalf("promote draft A: %v", err)
	}

	if _, _, _, err := releaseRepo.ArchivePreviousReleased(ctxB, docBID); err != nil {
		t.Fatalf("archive previous released B: %v", err)
	}
	if err := releaseRepo.PromoteDraftToReleased(ctxB, draftBID, []byte("draft-docx-B"), "approver-B"); err != nil {
		t.Fatalf("promote draft B: %v", err)
	}
	if err := releaseRepo.StoreRevisionDiff(ctxB, draftBID, json.RawMessage(`{"added":[],"removed":[],"modified":[]}`)); err != nil {
		t.Fatalf("store revision diff B: %v", err)
	}
	if err := releaseRepo.DeleteImageRefs(ctxB, releasedBID); err != nil {
		t.Fatalf("delete image refs B: %v", err)
	}
	if err := releaseRepo.CleanupOrphanImages(ctxB); err != nil {
		t.Fatalf("cleanup orphan images B: %v", err)
	}

	if err := releaseRepo.StoreRevisionDiff(ctxA, draftAID, json.RawMessage(`{`)); err == nil {
		t.Fatal("expected invalid revision diff in context A to fail")
	}

	releasedAState := mustLoadReleaseVersionState(t, rootCtx, db, releasedAID)
	if releasedAState.status != "released" {
		t.Fatalf("released A status = %s, want released", releasedAState.status)
	}
	draftAState := mustLoadReleaseVersionState(t, rootCtx, db, draftAID)
	if draftAState.status != "draft" {
		t.Fatalf("draft A status = %s, want draft", draftAState.status)
	}
	if draftAState.approvedBy.Valid {
		t.Fatalf("draft A approved_by should be null, got %s", draftAState.approvedBy.String)
	}

	releasedBState := mustLoadReleaseVersionState(t, rootCtx, db, releasedBID)
	if releasedBState.status != "archived" {
		t.Fatalf("released B status = %s, want archived", releasedBState.status)
	}
	draftBState := mustLoadReleaseVersionState(t, rootCtx, db, draftBID)
	if draftBState.status != "released" {
		t.Fatalf("draft B status = %s, want released", draftBState.status)
	}
	if !draftBState.approvedBy.Valid || draftBState.approvedBy.String != "approver-B" {
		t.Fatalf("draft B approved_by = %+v, want approver-B", draftBState.approvedBy)
	}
}

type releaseVersionState struct {
	status        string
	contentBlocks []byte
	docxBytes     []byte
	revisionDiff  []byte
	approvedBy    sql.NullString
}

type releaseRepoTestContextKey string

func mustLoadReleaseVersionState(t *testing.T, ctx context.Context, db *sql.DB, versionID uuid.UUID) releaseVersionState {
	t.Helper()

	var state releaseVersionState
	if err := db.QueryRowContext(ctx, `
		SELECT status, content_blocks, docx_bytes, revision_diff, approved_by
		FROM metaldocs.document_versions_mddm
		WHERE id = $1
	`, versionID).Scan(&state.status, &state.contentBlocks, &state.docxBytes, &state.revisionDiff, &state.approvedBy); err != nil {
		t.Fatalf("load release version state %s: %v", versionID, err)
	}

	return state
}

func jsonEqual(left, right []byte) bool {
	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false
	}

	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false
	}

	return bytes.Equal(mustMarshalJSON(leftValue), mustMarshalJSON(rightValue))
}

func mustMarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func seedReleaseRepoDocument(t *testing.T, ctx context.Context, db *sql.DB, suffix string) (string, uuid.UUID, uuid.UUID) {
	t.Helper()

	docID := newTestDocument(t, db)
	mddmRepo := NewMDDMRepository(db)

	releasedContent := json.RawMessage(`{"mddm_version":1,"blocks":[{"id":"old-` + suffix + `"}],"template_ref":null}`)
	releasedTemplateRef := json.RawMessage(`{"template_id":"tpl-old-` + suffix + `","version":1}`)

	var releasedID uuid.UUID
	if err := db.QueryRowContext(ctx, `
		INSERT INTO metaldocs.document_versions_mddm (
			document_id, version_number, revision_label, status,
			content_blocks, docx_bytes, template_ref, content_hash, created_by, approved_at, approved_by
		)
		VALUES ($1, $2, $3, 'released', $4, $5, $6, $7, $8, now(), $9)
		RETURNING id
	`, docID, 1, "REV00-"+suffix, releasedContent, []byte("released-docx-"+suffix), releasedTemplateRef, "released-hash-"+suffix, "creator-"+suffix, "approver-"+suffix).Scan(&releasedID); err != nil {
		t.Fatalf("insert released version %s: %v", suffix, err)
	}

	draftID, err := mddmRepo.InsertDraft(ctx, InsertDraftParams{
		DocumentID:    docID,
		VersionNumber: 2,
		RevisionLabel: "REV01-" + suffix,
		ContentBlocks: json.RawMessage(`{"mddm_version":1,"blocks":[{"id":"draft-` + suffix + `"}],"template_ref":null}`),
		ContentHash:   "draft-hash-" + suffix,
		TemplateRef:   json.RawMessage(`{"template_id":"tpl-draft-` + suffix + `","version":2}`),
		CreatedBy:     "creator-draft-" + suffix,
	})
	if err != nil {
		t.Fatalf("insert draft %s: %v", suffix, err)
	}

	return docID, releasedID, draftID
}
