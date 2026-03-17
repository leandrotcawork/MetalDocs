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
		DocumentID:   "doc-1",
		Title:        "Contract",
		DocumentType: "contract",
		OwnerID:      "user-1",
		BusinessUnit: "legal",
		Department:   "contracts",
		MetadataJSON: map[string]any{
			"counterparty":    "Metal Nobre",
			"contract_number": "CNT-001",
			"start_date":      "2026-03-01",
			"end_date":        "2026-12-31",
		},
		InitialContent: "v1",
		TraceID:        "trace-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Status != domain.StatusDraft {
		t.Fatalf("expected status %s, got %s", domain.StatusDraft, doc.Status)
	}
	if doc.DocumentType != "contract" {
		t.Fatalf("expected document type contract, got %s", doc.DocumentType)
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
		DocumentID:   "doc-2",
		Title:        "Policy",
		DocumentType: "policy",
		OwnerID:      "user-2",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"policy_code": "POL-001",
		},
		InitialContent: "first",
		TraceID:        "trace-2",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	version, err := svc.AddVersion(context.Background(), domain.AddVersionCommand{
		DocumentID:    "doc-2",
		Content:       "second",
		ChangeSummary: "policy updated",
		TraceID:       "trace-3",
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
		DocumentID:   "doc-a",
		Title:        "A",
		DocumentType: "manual",
		OwnerID:      "user-a",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-001",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-b",
		Title:        "B",
		DocumentType: "report",
		OwnerID:      "user-b",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"report_period": "2026-Q1",
		},
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

func TestCreateDocumentRejectsUnknownType(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-invalid",
		Title:        "Invalid",
		DocumentType: "unknown_type",
		OwnerID:      "user-1",
		BusinessUnit: "ops",
		Department:   "general",
	})
	if err == nil {
		t.Fatal("expected invalid document type error")
	}
}

func TestCreateDocumentRejectsInvalidMetadataForType(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-metadata",
		Title:        "Invalid Contract",
		DocumentType: "contract",
		OwnerID:      "user-1",
		BusinessUnit: "legal",
		Department:   "contracts",
		MetadataJSON: map[string]any{
			"counterparty": "Metal Nobre",
		},
	})
	if err == nil {
		t.Fatal("expected invalid metadata error")
	}
}

func TestDiffVersionsDetectsContentChange(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-diff",
		Title:        "Manual",
		DocumentType: "manual",
		OwnerID:      "user-1",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-002",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = svc.AddVersion(context.Background(), domain.AddVersionCommand{
		DocumentID:    "doc-diff",
		Content:       "v2",
		ChangeSummary: "changed body",
	})
	if err != nil {
		t.Fatalf("unexpected add version error: %v", err)
	}

	diff, err := svc.DiffVersions(context.Background(), "doc-diff", 1, 2)
	if err != nil {
		t.Fatalf("unexpected diff error: %v", err)
	}
	if !diff.ContentChanged {
		t.Fatal("expected contentChanged to be true")
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
