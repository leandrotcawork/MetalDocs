package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

// ObsoleteService marks a document as obsolete (end-of-life).
type ObsoleteService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}

// ErrInvalidObsoleteSource is returned when the document is not in a state
// that permits an → obsolete transition (must be "published" or "superseded").
var ErrInvalidObsoleteSource = errors.New("approval: document must be in 'published' or 'superseded' state to mark obsolete")

// MarkObsoleteRequest carries all inputs for ObsoleteService.MarkObsolete.
type MarkObsoleteRequest struct {
	TenantID        string
	DocumentID      string
	MarkedBy        string // user_id
	RevisionVersion int    // OCC guard
	Reason          string
}

// MarkObsoleteResult is returned on a successful obsolete transition.
type MarkObsoleteResult struct {
	PriorStatus string // the document's status before the transition
}

// MarkObsolete transitions a document from "published" or "superseded" to
// "obsolete" and cancels any in-progress approval instance for that document.
// All writes occur within a single transaction (outbox pattern).
func (s *ObsoleteService) MarkObsolete(ctx context.Context, db *sql.DB, req MarkObsoleteRequest) (MarkObsoleteResult, error) {
	// Step 1: begin transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: begin tx: %w", err)
	}

	// Step 2: fetch current status + revision_version under a row-level lock.
	var priorStatus string
	var currentRevision int
	var areaCode string
	err = tx.QueryRowContext(ctx, `
		SELECT status, revision_version, area_code
		  FROM documents
		 WHERE id        = $1
		   AND tenant_id = $2
		 FOR UPDATE`,
		req.DocumentID, req.TenantID,
	).Scan(&priorStatus, &currentRevision, &areaCode)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return MarkObsoleteResult{}, repository.ErrNoActiveInstance
		}
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: load document: %w", err)
	}

	// Step 3: guard — only published or superseded may transition to obsolete.
	if priorStatus != "published" && priorStatus != "superseded" {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, ErrInvalidObsoleteSource
	}

	if err := setAuthzGUC(ctx, tx, req.TenantID, req.MarkedBy); err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: %w", err)
	}
	if err := authz.Require(ctx, tx, "doc.obsolete", areaCode); err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, err
	}

	// Step 4: OCC UPDATE — atomically set status and bump revision_version.
	res, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status           = 'obsolete',
		       revision_version = revision_version + 1
		 WHERE id               = $1
		   AND tenant_id        = $2
		   AND status           = $3
		   AND revision_version = $4`,
		req.DocumentID, req.TenantID, priorStatus, req.RevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: update document: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: rows affected: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, repository.ErrStaleRevision
	}

	// Step 5: cancel any in-progress approval instance (no error if none exist).
	_, err = tx.ExecContext(ctx, `
		UPDATE approval_instances
		   SET status       = 'cancelled',
		       completed_at = now()
		 WHERE document_v2_id = $1
		   AND status         = 'in_progress'`,
		req.DocumentID,
	)
	if err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: cancel approval instance: %w", err)
	}

	// Step 6: emit "document_obsoleted" governance event.
	payloadMap := map[string]any{
		"reason":       req.Reason,
		"prior_status": priorStatus,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: marshal event payload: %w", err)
	}
	event := GovernanceEvent{
		TenantID:     req.TenantID,
		EventType:    "document_obsoleted",
		ActorUserID:  req.MarkedBy,
		ResourceType: "document",
		ResourceID:   req.DocumentID,
		Reason:       req.Reason,
		PayloadJSON:  json.RawMessage(payloadBytes),
		OccurredAt:   s.clock.Now(),
	}
	if err := s.emitter.Emit(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: emit event: %w", err)
	}

	// Step 7: commit.
	if err := tx.Commit(); err != nil {
		return MarkObsoleteResult{}, fmt.Errorf("markObsolete: commit: %w", err)
	}

	return MarkObsoleteResult{PriorStatus: priorStatus}, nil
}
