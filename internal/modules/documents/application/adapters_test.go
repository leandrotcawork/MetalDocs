package application

import (
	"context"
	"testing"
	"time"

	docdomain "metaldocs/internal/modules/documents/domain"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
	workflowmemory "metaldocs/internal/modules/workflow/infrastructure/memory"
)

type stubUserDisplayNameResolver struct{}

func (stubUserDisplayNameResolver) ResolveDisplayName(context.Context, string) (string, error) {
	return "Example User", nil
}

func TestServiceWithResolversStoresDependencies(t *testing.T) {
	service := NewService(nil, nil, nil)
	userResolver := stubUserDisplayNameResolver{}
	approvalReader := &WorkflowApprovalAdapter{}

	got := service.
		WithUserResolver(userResolver).
		WithApprovalReader(approvalReader)

	if got != service {
		t.Fatalf("expected builder methods to return the same service instance")
	}
	if service.userResolver != userResolver {
		t.Fatalf("expected user resolver to be stored on service")
	}
	if service.approvalReader != approvalReader {
		t.Fatalf("expected approval reader to be stored on service")
	}
}

func TestWorkflowApprovalAdapterListApprovalsMapsWorkflowApprovals(t *testing.T) {
	repo := workflowmemory.NewApprovalRepository()
	adapter := NewWorkflowApprovalAdapter(repo)
	decidedAt := time.Date(2026, time.April, 2, 15, 4, 5, 0, time.UTC)

	err := repo.Create(context.Background(), workflowdomain.Approval{
		ID:               "approval-1",
		DocumentID:       "doc-1",
		RequestedBy:      "requester-1",
		AssignedReviewer: "reviewer-1",
		DecisionBy:       "approver-1",
		Status:           workflowdomain.ApprovalStatusApproved,
		RequestedAt:      decidedAt.Add(-time.Hour),
		DecidedAt:        &decidedAt,
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	approvals, err := adapter.ListApprovals(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("list approvals: %v", err)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected 1 approval summary, got %d", len(approvals))
	}

	want := docdomain.ApprovalSummary{
		ApproverID: "approver-1",
		Status:     workflowdomain.ApprovalStatusApproved,
		DecidedAt:  &decidedAt,
	}
	if approvals[0] != want {
		t.Fatalf("unexpected approval summary: got %+v want %+v", approvals[0], want)
	}
}
