package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

// CancelService cancels an in-progress approval instance and reverts the
// document back to draft status.
type CancelService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}

// ErrReasonRequired is returned when CancelInput.Reason is empty.
var ErrReasonRequired = errors.New("cancel: reason must not be empty")

// CancelInput carries all inputs for CancelService.CancelInstance.
type CancelInput struct {
	TenantID                string
	InstanceID              string
	ExpectedRevisionVersion int    // OCC guard on the document
	ActorUserID             string
	Reason                  string
	// BypassAuthz, when true, sets metaldocs.bypass_authz inside the cancel
	// transaction before the authz check. Used by system jobs (e.g. watchdog)
	// that must act without a user capability claim.
	BypassAuthz bool
}

// CancelResult is returned on a successful cancellation.
type CancelResult struct {
	DocumentID string
}

// CancelInstance cancels an in-progress approval instance, transitions all
// active/pending stages to cancelled, and reverts the document to draft.
// Requires the workflow.instance.cancel capability for the document's area.
func (s *CancelService) CancelInstance(ctx context.Context, db *sql.DB, in CancelInput) (CancelResult, error) {
	// Guard: reason required.
	if in.Reason == "" {
		return CancelResult{}, ErrReasonRequired
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return CancelResult{}, fmt.Errorf("cancel: begin tx: %w", err)
	}

	// Load instance to get document ID and verify not already terminal.
	inst, err := s.repo.LoadInstance(ctx, tx, in.TenantID, in.InstanceID)
	if err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: load instance: %w", err)
	}
	if inst == nil {
		tx.Rollback()
		return CancelResult{}, repository.ErrNoActiveInstance
	}
	if inst.Status != domain.InstanceInProgress {
		tx.Rollback()
		return CancelResult{}, repository.ErrInstanceCompleted
	}

	docID := inst.DocumentID

	// Fetch document area_code for authz check. FOR UPDATE locks the document row
	// to prevent concurrent area_code changes between authz decision and status update.
	var areaCode string
	err = tx.QueryRowContext(ctx,
		`SELECT process_area_code_snapshot FROM documents WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
		docID, in.TenantID,
	).Scan(&areaCode)
	if err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: fetch area_code: %w", err)
	}

	// If caller is a system job bypassing user authz, set GUC inside this tx.
	if in.BypassAuthz {
		if _, err = tx.ExecContext(ctx,
			`SELECT set_config('metaldocs.bypass_authz', 'system', true)`,
		); err != nil {
			tx.Rollback()
			return CancelResult{}, fmt.Errorf("cancel: set bypass_authz GUC: %w", err)
		}
	} else {
		if err = setAuthzGUC(ctx, tx, in.TenantID, in.ActorUserID); err != nil {
			tx.Rollback()
			return CancelResult{}, fmt.Errorf("cancel: %w", err)
		}
	}

	// Authz gate: require workflow.instance.cancel capability.
	ctx = authz.WithCapCache(ctx)
	if err := authz.Require(ctx, tx, "workflow.instance.cancel", areaCode); err != nil {
		tx.Rollback()
		return CancelResult{}, err
	}

	// SET LOCAL cancel GUC — authorises under_review→draft transition in trigger.
	if _, err = tx.ExecContext(ctx,
		`SELECT set_config('metaldocs.cancel_in_progress', $1, true)`,
		in.InstanceID,
	); err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: set cancel GUC: %w", err)
	}

	// Cancel approval instance.
	now := s.clock.Now()
	if err = s.repo.UpdateInstanceStatus(ctx, tx, in.TenantID, in.InstanceID,
		domain.InstanceCancelled, domain.InstanceInProgress, &now); err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: update instance status: %w", err)
	}

	// Cancel all active and pending stage instances.
	if _, err = tx.ExecContext(ctx, `
		UPDATE approval_stage_instances
		   SET status = 'cancelled'
		 WHERE approval_instance_id = $1
		   AND status IN ('active','pending')`,
		in.InstanceID,
	); err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: cancel stages: %w", err)
	}

	// Revert document to draft (trigger enforces under_review→draft only with GUC set).
	res, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status           = 'draft',
		       revision_version = revision_version + 1
		 WHERE id               = $1
		   AND tenant_id        = $2
		   AND status           = 'under_review'
		   AND revision_version = $3`,
		docID, in.TenantID, in.ExpectedRevisionVersion,
	)
	if err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: revert doc to draft: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: rows affected: %w", err)
	}
	if rows == 0 {
		tx.Rollback()
		return CancelResult{}, repository.ErrStaleRevision
	}

	// Emit governance event.
	payload, _ := json.Marshal(map[string]any{
		"instance_id": in.InstanceID,
		"reason":      in.Reason,
	})
	if err = s.emitter.Emit(ctx, tx, GovernanceEvent{
		TenantID:     in.TenantID,
		EventType:    "approval.instance_cancelled",
		ActorUserID:  in.ActorUserID,
		ResourceType: "document",
		ResourceID:   docID,
		Reason:       in.Reason,
		PayloadJSON:  payload,
		OccurredAt:   now,
	}); err != nil {
		tx.Rollback()
		return CancelResult{}, fmt.Errorf("cancel: emit event: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return CancelResult{}, fmt.Errorf("cancel: commit: %w", err)
	}
	return CancelResult{DocumentID: docID}, nil
}

// newCancelService constructs a CancelService (wired by NewServices).
func newCancelService(repo repository.ApprovalRepository, emitter EventEmitter, clock Clock) *CancelService {
	return &CancelService{repo: repo, emitter: emitter, clock: clock}
}

