package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

type ApprovalRepository struct {
	db *sql.DB
}

func NewApprovalRepository(db *sql.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

func (r *ApprovalRepository) Create(ctx context.Context, approval workflowdomain.Approval) error {
	const q = `
INSERT INTO metaldocs.workflow_approvals (
  id, document_id, requested_by, assigned_reviewer, decision_by, status,
  request_reason, decision_reason, requested_at, decided_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`
	_, err := r.db.ExecContext(ctx, q,
		approval.ID,
		approval.DocumentID,
		approval.RequestedBy,
		approval.AssignedReviewer,
		nullIfEmpty(approval.DecisionBy),
		approval.Status,
		approval.RequestReason,
		nullIfEmpty(approval.DecisionReason),
		approval.RequestedAt,
		approval.DecidedAt,
	)
	if err != nil {
		return fmt.Errorf("create workflow approval: %w", err)
	}
	return nil
}

func (r *ApprovalRepository) GetLatestByDocumentID(ctx context.Context, documentID string) (workflowdomain.Approval, error) {
	const q = `
SELECT id, document_id, requested_by, assigned_reviewer, decision_by, status,
       request_reason, decision_reason, requested_at, decided_at
FROM metaldocs.workflow_approvals
WHERE document_id = $1
ORDER BY requested_at DESC
LIMIT 1
`
	var approval workflowdomain.Approval
	var decisionBy sql.NullString
	var decisionReason sql.NullString
	var decidedAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(documentID)).Scan(
		&approval.ID,
		&approval.DocumentID,
		&approval.RequestedBy,
		&approval.AssignedReviewer,
		&decisionBy,
		&approval.Status,
		&approval.RequestReason,
		&decisionReason,
		&approval.RequestedAt,
		&decidedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return workflowdomain.Approval{}, workflowdomain.ErrApprovalNotFound
		}
		return workflowdomain.Approval{}, fmt.Errorf("get latest workflow approval: %w", err)
	}
	if decisionBy.Valid {
		approval.DecisionBy = decisionBy.String
	}
	if decisionReason.Valid {
		approval.DecisionReason = decisionReason.String
	}
	if decidedAt.Valid {
		decidedUTC := decidedAt.Time.UTC()
		approval.DecidedAt = &decidedUTC
	}
	return approval, nil
}

func (r *ApprovalRepository) UpdateDecision(ctx context.Context, approvalID, status, decisionBy, decisionReason string, decidedAt time.Time) error {
	const q = `
UPDATE metaldocs.workflow_approvals
SET status = $2, decision_by = $3, decision_reason = $4, decided_at = $5
WHERE id = $1
`
	res, err := r.db.ExecContext(ctx, q, strings.TrimSpace(approvalID), strings.TrimSpace(status), nullIfEmpty(decisionBy), nullIfEmpty(decisionReason), decidedAt.UTC())
	if err != nil {
		return fmt.Errorf("update workflow approval decision: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected update workflow approval decision: %w", err)
	}
	if affected == 0 {
		return workflowdomain.ErrApprovalNotFound
	}
	return nil
}

func (r *ApprovalRepository) SaveState(ctx context.Context, approval workflowdomain.Approval) error {
	const q = `
UPDATE metaldocs.workflow_approvals
SET requested_by = $2, assigned_reviewer = $3, decision_by = $4, status = $5,
    request_reason = $6, decision_reason = $7, requested_at = $8, decided_at = $9
WHERE id = $1
`
	res, err := r.db.ExecContext(ctx, q,
		strings.TrimSpace(approval.ID),
		approval.RequestedBy,
		approval.AssignedReviewer,
		nullIfEmpty(approval.DecisionBy),
		approval.Status,
		approval.RequestReason,
		nullIfEmpty(approval.DecisionReason),
		approval.RequestedAt.UTC(),
		approval.DecidedAt,
	)
	if err != nil {
		return fmt.Errorf("save workflow approval state: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected save workflow approval state: %w", err)
	}
	if affected == 0 {
		return workflowdomain.ErrApprovalNotFound
	}
	return nil
}

func (r *ApprovalRepository) Delete(ctx context.Context, approvalID string) error {
	const q = `DELETE FROM metaldocs.workflow_approvals WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, strings.TrimSpace(approvalID))
	if err != nil {
		return fmt.Errorf("delete workflow approval: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected delete workflow approval: %w", err)
	}
	if affected == 0 {
		return workflowdomain.ErrApprovalNotFound
	}
	return nil
}

func (r *ApprovalRepository) ListByDocumentID(ctx context.Context, documentID string) ([]workflowdomain.Approval, error) {
	const q = `
SELECT id, document_id, requested_by, assigned_reviewer, decision_by, status,
       request_reason, decision_reason, requested_at, decided_at
FROM metaldocs.workflow_approvals
WHERE document_id = $1
ORDER BY requested_at ASC
`
	rows, err := r.db.QueryContext(ctx, q, strings.TrimSpace(documentID))
	if err != nil {
		return nil, fmt.Errorf("list workflow approvals: %w", err)
	}
	defer rows.Close()

	var out []workflowdomain.Approval
	for rows.Next() {
		var approval workflowdomain.Approval
		var decisionBy sql.NullString
		var decisionReason sql.NullString
		var decidedAt sql.NullTime
		if err := rows.Scan(
			&approval.ID,
			&approval.DocumentID,
			&approval.RequestedBy,
			&approval.AssignedReviewer,
			&decisionBy,
			&approval.Status,
			&approval.RequestReason,
			&decisionReason,
			&approval.RequestedAt,
			&decidedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workflow approval: %w", err)
		}
		if decisionBy.Valid {
			approval.DecisionBy = decisionBy.String
		}
		if decisionReason.Valid {
			approval.DecisionReason = decisionReason.String
		}
		if decidedAt.Valid {
			decidedUTC := decidedAt.Time.UTC()
			approval.DecidedAt = &decidedUTC
		}
		out = append(out, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list workflow approvals rows: %w", err)
	}
	return out, nil
}

func nullIfEmpty(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

