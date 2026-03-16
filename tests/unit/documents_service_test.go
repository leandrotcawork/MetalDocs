package unit

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	"metaldocs/internal/platform/messaging"
)

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time {
	return f.now
}

type capturePublisher struct {
	events []messaging.Event
}

func (p *capturePublisher) Publish(_ context.Context, event messaging.Event) error {
	p.events = append(p.events, event)
	return nil
}

func TestCreateDocumentCreatesVersionAndEvents(t *testing.T) {
	repo := memory.NewRepository()
	pub := &capturePublisher{}
	svc := application.NewService(repo, pub, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	doc, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:     "doc-1",
		Title:          "Contract",
		OwnerID:        "user-1",
		InitialContent: "v1",
		TraceID:        "trace-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Status != domain.StatusDraft {
		t.Fatalf("expected status %s, got %s", domain.StatusDraft, doc.Status)
	}

	versions, err := svc.ListVersions(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	if versions[0].Number != 1 {
		t.Fatalf("expected version 1, got %d", versions[0].Number)
	}

	if len(pub.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(pub.events))
	}
}

func TestAddVersionIncrementsVersionNumber(t *testing.T) {
	repo := memory.NewRepository()
	pub := &capturePublisher{}
	svc := application.NewService(repo, pub, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:     "doc-2",
		Title:          "Policy",
		OwnerID:        "user-2",
		InitialContent: "first",
		TraceID:        "trace-2",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	version, err := svc.AddVersion(context.Background(), domain.AddVersionCommand{
		DocumentID: "doc-2",
		Content:    "second",
		TraceID:    "trace-3",
	})
	if err != nil {
		t.Fatalf("unexpected add version error: %v", err)
	}

	if version.Number != 2 {
		t.Fatalf("expected version 2, got %d", version.Number)
	}

	versions, err := svc.ListVersions(context.Background(), "doc-2")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}

func TestListDocumentsReturnsCreatedDocuments(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:     "doc-a",
		Title:          "A",
		OwnerID:        "user-a",
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:     "doc-b",
		Title:          "B",
		OwnerID:        "user-b",
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	docs, err := svc.ListDocuments(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}
}

func TestCreateDocumentValidation(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestListVersionsRequiresExistingDocument(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	_, err := svc.ListVersions(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing document")
	}
}
