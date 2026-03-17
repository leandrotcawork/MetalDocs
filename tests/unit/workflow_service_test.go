package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	auditmemory "metaldocs/internal/modules/audit/infrastructure/memory"
	docapp "metaldocs/internal/modules/documents/application"
	docdomain "metaldocs/internal/modules/documents/domain"
	docmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

type failingAuditWriter struct{}

func (f failingAuditWriter) Record(context.Context, auditdomain.Event) error {
	return errors.New("audit write failed")
}

func TestWorkflowTransitionUpdatesDocumentStatus(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, auditmemory.NewWriter(), nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "doc-wf-1",
		Title:        "Workflow Doc",
		DocumentType: "manual",
		OwnerID:      "owner-1",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-WF-1",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	result, err := workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID:       "doc-wf-1",
		ToStatus:         docdomain.StatusInReview,
		ActorID:          "owner-1",
		AssignedReviewer: "reviewer-user",
		Reason:           "ready for review",
		TraceID:          "trace-wf-1",
	})
	if err != nil {
		t.Fatalf("unexpected transition error: %v", err)
	}

	if result.FromStatus != docdomain.StatusDraft || result.ToStatus != docdomain.StatusInReview {
		t.Fatalf("unexpected transition result: %+v", result)
	}
	if result.AssignedReviewer != "reviewer-user" {
		t.Fatalf("expected assigned reviewer reviewer-user, got %s", result.AssignedReviewer)
	}

	doc, err := repo.GetDocument(context.Background(), "doc-wf-1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if doc.Status != docdomain.StatusInReview {
		t.Fatalf("expected status %s, got %s", docdomain.StatusInReview, doc.Status)
	}

	approvals, err := workflowSvc.ListApprovals(context.Background(), "doc-wf-1")
	if err != nil {
		t.Fatalf("unexpected list approvals error: %v", err)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(approvals))
	}
	if approvals[0].Status != workflowdomain.ApprovalStatusPending {
		t.Fatalf("expected pending approval, got %s", approvals[0].Status)
	}
}

func TestWorkflowTransitionRejectsInvalidPath(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, auditmemory.NewWriter(), nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "doc-wf-2",
		Title:        "Workflow Doc 2",
		DocumentType: "manual",
		OwnerID:      "owner-2",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-WF-2",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID: "doc-wf-2",
		ToStatus:   docdomain.StatusPublished,
		ActorID:    "reviewer-user",
	})
	if !errors.Is(err, workflowdomain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestWorkflowApprovalRequiresAssignedReviewerOwnership(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, auditmemory.NewWriter(), nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "doc-wf-ownership",
		Title:          "Workflow Ownership",
		DocumentType:   "manual",
		OwnerID:        "owner-4",
		BusinessUnit:   "ops",
		Department:     "general",
		MetadataJSON:   map[string]any{"manual_code": "MAN-WF-4"},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID:       "doc-wf-ownership",
		ToStatus:         docdomain.StatusInReview,
		ActorID:          "owner-4",
		AssignedReviewer: "reviewer-assigned",
		Reason:           "submit for approval",
	})
	if err != nil {
		t.Fatalf("unexpected request review error: %v", err)
	}

	_, err = workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID: "doc-wf-ownership",
		ToStatus:   docdomain.StatusApproved,
		ActorID:    "reviewer-other",
		Reason:     "approving",
	})
	if !errors.Is(err, workflowdomain.ErrApprovalReviewerDenied) {
		t.Fatalf("expected ErrApprovalReviewerDenied, got %v", err)
	}
}

func TestWorkflowApprovalApprovesAndRecordsDecision(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, auditmemory.NewWriter(), nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "doc-wf-approve",
		Title:          "Workflow Approval",
		DocumentType:   "manual",
		OwnerID:        "owner-5",
		BusinessUnit:   "ops",
		Department:     "general",
		MetadataJSON:   map[string]any{"manual_code": "MAN-WF-5"},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID:       "doc-wf-approve",
		ToStatus:         docdomain.StatusInReview,
		ActorID:          "owner-5",
		AssignedReviewer: "reviewer-approved",
		Reason:           "submit for approval",
	})
	if err != nil {
		t.Fatalf("unexpected request review error: %v", err)
	}

	result, err := workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID: "doc-wf-approve",
		ToStatus:   docdomain.StatusApproved,
		ActorID:    "reviewer-approved",
		Reason:     "looks good",
	})
	if err != nil {
		t.Fatalf("unexpected approve error: %v", err)
	}
	if result.ApprovalStatus != workflowdomain.ApprovalStatusApproved {
		t.Fatalf("expected approved status, got %s", result.ApprovalStatus)
	}

	approvals, err := workflowSvc.ListApprovals(context.Background(), "doc-wf-approve")
	if err != nil {
		t.Fatalf("unexpected list approvals error: %v", err)
	}
	if len(approvals) != 1 {
		t.Fatalf("expected 1 approval, got %d", len(approvals))
	}
	if approvals[0].DecisionBy != "reviewer-approved" {
		t.Fatalf("expected decision by reviewer-approved, got %s", approvals[0].DecisionBy)
	}
	if approvals[0].DecisionReason != "looks good" {
		t.Fatalf("expected decision reason to be recorded, got %s", approvals[0].DecisionReason)
	}
}

func TestWorkflowTransitionRollsBackWhenAuditFails(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, failingAuditWriter{}, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "doc-wf-3",
		Title:        "Workflow Doc 3",
		DocumentType: "manual",
		OwnerID:      "owner-3",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-WF-3",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID:       "doc-wf-3",
		ToStatus:         docdomain.StatusInReview,
		ActorID:          "owner-3",
		AssignedReviewer: "reviewer-user",
		Reason:           "ready for review",
	})
	if err == nil {
		t.Fatal("expected transition error when audit fails")
	}

	doc, err := repo.GetDocument(context.Background(), "doc-wf-3")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if doc.Status != docdomain.StatusDraft {
		t.Fatalf("expected status rollback to %s, got %s", docdomain.StatusDraft, doc.Status)
	}
}
