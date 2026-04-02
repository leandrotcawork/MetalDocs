package application

import (
	"context"

	"metaldocs/internal/modules/documents/domain"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

// WorkflowApprovalAdapter adapts workflow.ApprovalRepository to documents.WorkflowApprovalReader.
type WorkflowApprovalAdapter struct {
	repo workflowdomain.ApprovalRepository
}

func NewWorkflowApprovalAdapter(repo workflowdomain.ApprovalRepository) *WorkflowApprovalAdapter {
	return &WorkflowApprovalAdapter{repo: repo}
}

func (a *WorkflowApprovalAdapter) ListApprovals(ctx context.Context, documentID string) ([]domain.ApprovalSummary, error) {
	approvals, err := a.repo.ListByDocumentID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.ApprovalSummary, len(approvals))
	for i, approval := range approvals {
		out[i] = domain.ApprovalSummary{
			ApproverID: approval.DecisionBy,
			Status:     approval.Status,
			DecidedAt:  approval.DecidedAt,
		}
	}
	return out, nil
}
