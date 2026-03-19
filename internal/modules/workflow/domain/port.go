package domain

import (
	"context"
	"time"
)

// ApprovalRepository persists workflow approvals owned by the workflow module.
// It is intentionally separate from the documents repository to preserve storage boundaries.
type ApprovalRepository interface {
	Create(ctx context.Context, approval Approval) error
	GetLatestByDocumentID(ctx context.Context, documentID string) (Approval, error)
	UpdateDecision(ctx context.Context, approvalID, status, decisionBy, decisionReason string, decidedAt time.Time) error
	SaveState(ctx context.Context, approval Approval) error
	Delete(ctx context.Context, approvalID string) error
	ListByDocumentID(ctx context.Context, documentID string) ([]Approval, error)
}
