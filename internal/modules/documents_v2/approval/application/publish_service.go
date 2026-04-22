package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// PublishService handles transitioning an approved document to published state.
type PublishService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}

// ErrInstanceNotApproved is returned when PublishApproved is called on an
// instance whose status is not "approved".
var ErrInstanceNotApproved = errors.New("approval: instance is not in approved state")

// PublishRequest carries all inputs for PublishApproved.
type PublishRequest struct {
	TenantID    string
	InstanceID  string
	PublishedBy string // user_id triggering publish
}

// PublishResult is returned on successful publish.
type PublishResult struct {
	DocumentID string
	NewStatus  string // "published"
}

// PublishApproved transitions an approved document to published state.
// It verifies the approval instance is in "approved" status, performs an OCC
// UPDATE on the documents table (approved → published), emits a
// "document_published" governance event, and commits.
func (s *PublishService) PublishApproved(ctx context.Context, db *sql.DB, req PublishRequest) (PublishResult, error) {
	// Step 1: begin transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return PublishResult{}, fmt.Errorf("publishApproved: begin tx: %w", err)
	}

	// Step 2: load the approval instance.
	instance, err := s.repo.LoadInstance(ctx, tx, req.TenantID, req.InstanceID)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return PublishResult{}, repository.ErrNoActiveInstance
		}
		return PublishResult{}, fmt.Errorf("publishApproved: load instance: %w", err)
	}
	if instance == nil {
		_ = tx.Rollback()
		return PublishResult{}, repository.ErrNoActiveInstance
	}

	// Verify instance is in approved state.
	if instance.Status != domain.InstanceApproved {
		_ = tx.Rollback()
		return PublishResult{}, ErrInstanceNotApproved
	}

	// Step 3: OCC transition the document from "approved" to "published".
	// Uses revision_version as the optimistic concurrency guard.
	result, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status = 'published'
		 WHERE id             = $1
		   AND tenant_id      = $2
		   AND status         = 'approved'
		   AND revision_version = $3`,
		instance.DocumentID, req.TenantID, instance.RevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return PublishResult{}, fmt.Errorf("publishApproved: update document state: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return PublishResult{}, fmt.Errorf("publishApproved: rows affected: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return PublishResult{}, repository.ErrStaleRevision
	}

	// Step 4: emit "document_published" governance event.
	now := s.clock.Now()
	payloadMap := map[string]any{
		"instance_id":      req.InstanceID,
		"revision_version": instance.RevisionVersion,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		_ = tx.Rollback()
		return PublishResult{}, fmt.Errorf("publishApproved: marshal event payload: %w", err)
	}
	event := GovernanceEvent{
		TenantID:     req.TenantID,
		EventType:    "document_published",
		ActorUserID:  req.PublishedBy,
		ResourceType: "document",
		ResourceID:   instance.DocumentID,
		PayloadJSON:  json.RawMessage(payloadBytes),
		OccurredAt:   now,
	}
	if err := s.emitter.Emit(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		return PublishResult{}, fmt.Errorf("publishApproved: emit event: %w", err)
	}

	// Step 5: commit.
	if err := tx.Commit(); err != nil {
		return PublishResult{}, fmt.Errorf("publishApproved: commit: %w", err)
	}

	return PublishResult{DocumentID: instance.DocumentID, NewStatus: "published"}, nil
}

// ErrEffectiveDateInPast is returned when SchedulePublish is called with an
// effective_date that is not strictly in the future.
var ErrEffectiveDateInPast = errors.New("approval: effective_date must be in the future")

// SchedulePublishRequest carries all inputs for SchedulePublish.
type SchedulePublishRequest struct {
	TenantID      string
	InstanceID    string
	EffectiveDate time.Time // must be strictly after clock.Now()
	ScheduledBy   string
}

// SchedulePublishResult is returned on successful scheduling.
type SchedulePublishResult struct {
	DocumentID    string
	EffectiveDate time.Time
}

// SchedulePublish transitions an approved document to "scheduled" state with a
// future effective date. It guards against past dates, performs an OCC UPDATE
// on the documents table (approved → scheduled), emits a "publish_scheduled"
// governance event, and commits.
func (s *PublishService) SchedulePublish(ctx context.Context, db *sql.DB, req SchedulePublishRequest) (SchedulePublishResult, error) {
	// Step 1: guard — effective_date must be strictly in the future.
	if !req.EffectiveDate.After(s.clock.Now()) {
		return SchedulePublishResult{}, ErrEffectiveDateInPast
	}

	// Step 2: begin transaction.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: begin tx: %w", err)
	}

	// Step 3: load the approval instance.
	instance, err := s.repo.LoadInstance(ctx, tx, req.TenantID, req.InstanceID)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return SchedulePublishResult{}, repository.ErrNoActiveInstance
		}
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: load instance: %w", err)
	}
	if instance == nil {
		_ = tx.Rollback()
		return SchedulePublishResult{}, repository.ErrNoActiveInstance
	}

	// Verify instance is in approved state.
	if instance.Status != domain.InstanceApproved {
		_ = tx.Rollback()
		return SchedulePublishResult{}, ErrInstanceNotApproved
	}

	// Step 4: OCC transition the document from "approved" to "scheduled".
	result, err := tx.ExecContext(ctx, `
		UPDATE documents
		   SET status           = 'scheduled',
		       effective_date   = $1,
		       revision_version = revision_version + 1
		 WHERE id               = $2
		   AND tenant_id        = $3
		   AND status           = 'approved'
		   AND revision_version = $4`,
		req.EffectiveDate.UTC(), instance.DocumentID, req.TenantID, instance.RevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: update document state: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: rows affected: %w", err)
	}
	if affected == 0 {
		_ = tx.Rollback()
		return SchedulePublishResult{}, repository.ErrStaleRevision
	}

	// Step 5: emit "publish_scheduled" governance event.
	now := s.clock.Now()
	payloadMap := map[string]any{
		"effective_date": req.EffectiveDate.UTC().Format(time.RFC3339),
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		_ = tx.Rollback()
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: marshal event payload: %w", err)
	}
	event := GovernanceEvent{
		TenantID:     req.TenantID,
		EventType:    "publish_scheduled",
		ActorUserID:  req.ScheduledBy,
		ResourceType: "document",
		ResourceID:   instance.DocumentID,
		PayloadJSON:  json.RawMessage(payloadBytes),
		OccurredAt:   now,
	}
	if err := s.emitter.Emit(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: emit event: %w", err)
	}

	// Step 6: commit.
	if err := tx.Commit(); err != nil {
		return SchedulePublishResult{}, fmt.Errorf("schedulePublish: commit: %w", err)
	}

	return SchedulePublishResult{DocumentID: instance.DocumentID, EffectiveDate: req.EffectiveDate}, nil
}
