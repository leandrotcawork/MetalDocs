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
		DocumentID:     "doc-wf-1",
		Title:          "Workflow Doc",
		OwnerID:        "owner-1",
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	result, err := workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID: "doc-wf-1",
		ToStatus:   docdomain.StatusInReview,
		ActorID:    "reviewer-user",
		Reason:     "ready for review",
		TraceID:    "trace-wf-1",
	})
	if err != nil {
		t.Fatalf("unexpected transition error: %v", err)
	}

	if result.FromStatus != docdomain.StatusDraft || result.ToStatus != docdomain.StatusInReview {
		t.Fatalf("unexpected transition result: %+v", result)
	}

	doc, err := repo.GetDocument(context.Background(), "doc-wf-1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if doc.Status != docdomain.StatusInReview {
		t.Fatalf("expected status %s, got %s", docdomain.StatusInReview, doc.Status)
	}
}

func TestWorkflowTransitionRejectsInvalidPath(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, auditmemory.NewWriter(), nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "doc-wf-2",
		Title:          "Workflow Doc 2",
		OwnerID:        "owner-2",
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

func TestWorkflowTransitionRollsBackWhenAuditFails(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	workflowSvc := workflowapp.NewService(repo, failingAuditWriter{}, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 1, 0, 0, time.UTC)})

	_, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "doc-wf-3",
		Title:          "Workflow Doc 3",
		OwnerID:        "owner-3",
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = workflowSvc.Transition(context.Background(), workflowdomain.TransitionCommand{
		DocumentID: "doc-wf-3",
		ToStatus:   docdomain.StatusInReview,
		ActorID:    "reviewer-user",
		Reason:     "ready for review",
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
