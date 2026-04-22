package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"metaldocs/internal/modules/documents_v2/approval/repository"
	"metaldocs/internal/modules/iam/authz"
)

// SchedulerService processes scheduled publish jobs (F6 — ListScheduledDue).
type SchedulerService struct {
	repo    repository.ApprovalRepository
	emitter EventEmitter
	clock   Clock
}

// RunDuePublishesResult summarises a scheduler batch run.
type RunDuePublishesResult struct {
	Processed int
	Errors    []error
}

const schedulerBatchLimit = 50

const updateScheduledDocSQL = `
UPDATE documents
   SET status = 'published',
       revision_version = revision_version + 1
 WHERE id = $1
   AND tenant_id = $2
   AND status = 'scheduled'
   AND revision_version = $3`

// RunDuePublishes fetches up to 50 rows whose effective_date <= now and
// status = 'scheduled', then publishes each one inside its own transaction.
// Per-row errors are collected in result.Errors; the method only returns a
// non-nil top-level error when the initial fetch itself fails.
func (s *SchedulerService) RunDuePublishes(ctx context.Context, db *sql.DB) (RunDuePublishesResult, error) {
	var result RunDuePublishesResult

	// Open a read-committed transaction for the SKIP LOCKED fetch.
	fetchTx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return result, fmt.Errorf("scheduler: begin fetch tx: %w", err)
	}

	rows, err := s.repo.ListScheduledDue(ctx, fetchTx, s.clock.Now(), schedulerBatchLimit)
	if err != nil {
		_ = fetchTx.Rollback()
		return result, fmt.Errorf("scheduler: fetch due publishes: %w", err)
	}

	// Release the fetch transaction — the rows are already in memory.
	if err = fetchTx.Commit(); err != nil {
		return result, fmt.Errorf("scheduler: commit fetch tx: %w", err)
	}

	for _, row := range rows {
		published, procErr := s.processRow(ctx, db, row)
		if procErr != nil {
			result.Errors = append(result.Errors, procErr)
		} else if published {
			result.Processed++
		}
	}

	return result, nil
}

// processRow publishes a single scheduled document inside its own transaction.
// It returns (true, nil) on successful publish, (false, nil) when another runner
// already published the row (RowsAffected == 0), and (false, err) on failure.
func (s *SchedulerService) processRow(ctx context.Context, db *sql.DB, row repository.ScheduledPublishRow) (bool, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("scheduler: begin publish tx for doc %s: %w", row.DocumentID, err)
	}
	if err := authz.BypassSystem(ctx, tx); err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("scheduler: bypass authz for doc %s: %w", row.DocumentID, err)
	}

	res, err := tx.ExecContext(ctx, updateScheduledDocSQL,
		row.DocumentID, row.TenantID, row.RevisionVersion,
	)
	if err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("scheduler: update document %s: %w", row.DocumentID, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("scheduler: rows affected for doc %s: %w", row.DocumentID, err)
	}

	if affected == 0 {
		// Another runner already published this document — skip cleanly.
		_ = tx.Commit()
		return false, nil
	}

	payload, _ := json.Marshal(map[string]any{
		"revision_version": row.RevisionVersion + 1,
		"effective_date":   row.EffectiveFrom.Format("2006-01-02T15:04:05Z"),
	})

	ev := GovernanceEvent{
		TenantID:     row.TenantID,
		EventType:    "document_published",
		ActorUserID:  "scheduler",
		ResourceType: "document",
		ResourceID:   row.DocumentID,
		Reason:       "scheduled publish",
		PayloadJSON:  json.RawMessage(payload),
	}

	if err = s.emitter.Emit(ctx, tx, ev); err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("scheduler: emit event for doc %s: %w", row.DocumentID, err)
	}

	if err = tx.Commit(); err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("scheduler: commit publish tx for doc %s: %w", row.DocumentID, err)
	}

	return true, nil
}
