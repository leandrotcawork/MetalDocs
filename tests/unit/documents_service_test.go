package unit

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
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
		Title:        "Procedure",
		DocumentType: "po",
		OwnerID:      "user-1",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-001",
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
	if doc.DocumentType != "po" {
		t.Fatalf("expected document type po, got %s", doc.DocumentType)
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
		Title:        "Procedure",
		DocumentType: "po",
		OwnerID:      "user-2",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-001",
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
		ChangeSummary: "procedure updated",
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
		DocumentType: "it",
		OwnerID:      "user-a",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"instruction_code": "IT-001",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-b",
		Title:        "B",
		DocumentType: "rg",
		OwnerID:      "user-b",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"record_code": "RG-001",
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
		Title:        "Invalid Procedure",
		DocumentType: "po",
		OwnerID:      "user-1",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"wrong_field": "bad",
		},
	})
	if err == nil {
		t.Fatal("expected invalid metadata error")
	}
}

func TestCreateDocumentWithMetalNobreProfileAndProcessArea(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	doc, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:      "doc-mn-po",
		Title:           "Procedimento de Marketplaces",
		DocumentProfile: "po",
		ProcessArea:     "marketplaces",
		OwnerID:         "user-1",
		BusinessUnit:    "commercial",
		Department:      "marketplaces",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-MKT-001",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.DocumentProfile != "po" {
		t.Fatalf("expected profile po, got %s", doc.DocumentProfile)
	}
	if doc.DocumentFamily != "procedure" {
		t.Fatalf("expected family procedure, got %s", doc.DocumentFamily)
	}
	if doc.ProcessArea != "marketplaces" {
		t.Fatalf("expected processArea marketplaces, got %s", doc.ProcessArea)
	}
	if doc.ProfileSchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", doc.ProfileSchemaVersion)
	}
}

func TestListDocumentProfilesIncludesMetalNobreRegistry(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	items, err := svc.ListDocumentProfiles(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := map[string]bool{}
	for _, item := range items {
		found[item.Code] = true
	}

	for _, code := range []string{"po", "it", "rg"} {
		if !found[code] {
			t.Fatalf("expected profile %s in registry", code)
		}
	}
}

func TestDiffVersionsDetectsContentChange(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-diff",
		Title:        "Instruction",
		DocumentType: "it",
		OwnerID:      "user-1",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"instruction_code": "IT-002",
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

func TestUploadAndListAttachmentsAuthorized(t *testing.T) {
	repo := memory.NewRepository()
	store := memory.NewAttachmentStore()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)}).WithAttachmentStore(store)

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-attach",
		Title:        "Instruction",
		DocumentType: "it",
		OwnerID:      "owner-1",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"instruction_code": "IT-ATTACH",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	if err := svc.ReplaceAccessPolicies(context.Background(), "document", "doc-attach", []domain.AccessPolicy{
		{SubjectType: domain.SubjectTypeUser, SubjectID: "editor-1", Capability: domain.CapabilityDocumentView, Effect: domain.PolicyEffectAllow},
		{SubjectType: domain.SubjectTypeUser, SubjectID: "editor-1", Capability: domain.CapabilityDocumentUploadAttachment, Effect: domain.PolicyEffectAllow},
	}); err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	ctx := iamdomain.WithAuthContext(context.Background(), "editor-1", []iamdomain.Role{iamdomain.RoleEditor})
	attachment, err := svc.UploadAttachmentAuthorized(ctx, domain.UploadAttachmentCommand{
		DocumentID:  "doc-attach",
		FileName:    "manual.txt",
		ContentType: "text/plain",
		Content:     []byte("attachment content"),
		TraceID:     "trace-attach",
	})
	if err != nil {
		t.Fatalf("unexpected upload error: %v", err)
	}

	items, err := svc.ListAttachmentsAuthorized(ctx, "doc-attach")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(items))
	}
	if items[0].ID != attachment.ID {
		t.Fatalf("expected attachment %s, got %s", attachment.ID, items[0].ID)
	}

	got, content, err := svc.OpenAttachmentContent(context.Background(), attachment.ID)
	if err != nil {
		t.Fatalf("unexpected open content error: %v", err)
	}
	if got.FileName != "manual.txt" {
		t.Fatalf("expected manual.txt, got %s", got.FileName)
	}
	if string(content) != "attachment content" {
		t.Fatalf("unexpected content: %s", string(content))
	}
}
