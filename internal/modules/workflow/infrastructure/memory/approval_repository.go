package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

type ApprovalRepository struct {
	mu        sync.RWMutex
	approvals map[string][]workflowdomain.Approval
}

func NewApprovalRepository() *ApprovalRepository {
	return &ApprovalRepository{approvals: map[string][]workflowdomain.Approval{}}
}

func (r *ApprovalRepository) Create(_ context.Context, approval workflowdomain.Approval) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.approvals[approval.DocumentID] = append(r.approvals[approval.DocumentID], approval)
	return nil
}

func (r *ApprovalRepository) GetLatestByDocumentID(_ context.Context, documentID string) (workflowdomain.Approval, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.approvals[documentID]
	if len(items) == 0 {
		return workflowdomain.Approval{}, workflowdomain.ErrApprovalNotFound
	}
	return items[len(items)-1], nil
}

func (r *ApprovalRepository) UpdateDecision(_ context.Context, approvalID, status, decisionBy, decisionReason string, decidedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for documentID, items := range r.approvals {
		for idx, item := range items {
			if item.ID != approvalID {
				continue
			}
			item.Status = status
			item.DecisionBy = decisionBy
			item.DecisionReason = decisionReason
			decidedUTC := decidedAt.UTC()
			item.DecidedAt = &decidedUTC
			items[idx] = item
			r.approvals[documentID] = items
			return nil
		}
	}
	return workflowdomain.ErrApprovalNotFound
}

func (r *ApprovalRepository) SaveState(_ context.Context, approval workflowdomain.Approval) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := r.approvals[approval.DocumentID]
	for idx, item := range items {
		if item.ID != approval.ID {
			continue
		}
		items[idx] = approval
		r.approvals[approval.DocumentID] = items
		return nil
	}
	return workflowdomain.ErrApprovalNotFound
}

func (r *ApprovalRepository) Delete(_ context.Context, approvalID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for documentID, items := range r.approvals {
		for idx, item := range items {
			if item.ID != approvalID {
				continue
			}
			r.approvals[documentID] = append(items[:idx], items[idx+1:]...)
			return nil
		}
	}
	return workflowdomain.ErrApprovalNotFound
}

func (r *ApprovalRepository) ListByDocumentID(_ context.Context, documentID string) ([]workflowdomain.Approval, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := append([]workflowdomain.Approval(nil), r.approvals[documentID]...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].RequestedAt.Before(items[j].RequestedAt)
	})
	return items, nil
}

