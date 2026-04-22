package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

// SupersedeService marks a published document as superseded by a newer revision.
type SupersedeService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}

// SupersedeRequest carries all inputs for PublishSuperseding.
type SupersedeRequest struct {
	TenantID             string
	NewDocumentID        string // the document being published (becomes "published")
	PriorDocumentID      string // the previous published document (becomes "superseded")
	SupersededBy         string // user_id
	NewRevisionVersion   int    // OCC for new doc
	PriorRevisionVersion int    // OCC for prior doc
}

// SupersedeResult is returned on successful publish-and-supersede.
type SupersedeResult struct {
	NewDocumentStatus   string // "published"
	PriorDocumentStatus string // "superseded"
}

// PublishSuperseding atomically transitions a new document from "approved" to
// "published" and the prior document from "published" to "superseded".
// Both OCC guards must pass; otherwise the transaction is rolled back and
// repository.ErrStaleRevision is returned.
func (s *SupersedeService) PublishSuperseding(ctx context.Context, db *sql.DB, req SupersedeRequest) (SupersedeResult, error) {
	// Step 1: begin transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: begin tx: %w", err)
	}

	areaCode, err := loadDocumentAreaCode(ctx, tx, req.TenantID, req.NewDocumentID)
	if err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: load document area: %w", err)
	}
	if err := authz.Require(ctx, tx, "doc.supersede", areaCode); err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, err
	}

	// Step 2: OCC UPDATE for new document (approved → published).
	newResult, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status           = 'published',
		       revision_version = revision_version + 1
		 WHERE id               = $1
		   AND tenant_id        = $2
		   AND status           = 'approved'
		   AND revision_version = $3`,
		req.NewDocumentID, req.TenantID, req.NewRevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: update new document: %w", err)
	}
	newAffected, err := newResult.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: rows affected (new): %w", err)
	}
	if newAffected == 0 {
		_ = tx.Rollback()
		return SupersedeResult{}, repository.ErrStaleRevision
	}

	// Step 3: OCC UPDATE for prior document (published → superseded).
	priorResult, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status           = 'superseded',
		       revision_version = revision_version + 1
		 WHERE id               = $1
		   AND tenant_id        = $2
		   AND status           = 'published'
		   AND revision_version = $3`,
		req.PriorDocumentID, req.TenantID, req.PriorRevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: update prior document: %w", err)
	}
	priorAffected, err := priorResult.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: rows affected (prior): %w", err)
	}
	if priorAffected == 0 {
		_ = tx.Rollback()
		return SupersedeResult{}, repository.ErrStaleRevision
	}

	// Step 4: emit "document_superseded" governance event.
	now := s.clock.Now()
	payloadMap := map[string]any{
		"new_document_id":   req.NewDocumentID,
		"prior_document_id": req.PriorDocumentID,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: marshal event payload: %w", err)
	}
	event := GovernanceEvent{
		TenantID:     req.TenantID,
		EventType:    "document_superseded",
		ActorUserID:  req.SupersededBy,
		ResourceType: "document",
		ResourceID:   req.NewDocumentID,
		PayloadJSON:  json.RawMessage(payloadBytes),
		OccurredAt:   now,
	}
	if err := s.emitter.Emit(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: emit event: %w", err)
	}

	// Step 5: commit.
	if err := tx.Commit(); err != nil {
		return SupersedeResult{}, fmt.Errorf("publishSuperseding: commit: %w", err)
	}

	return SupersedeResult{
		NewDocumentStatus:   "published",
		PriorDocumentStatus: "superseded",
	}, nil
}
