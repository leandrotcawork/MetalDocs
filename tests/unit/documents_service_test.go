package unit

import (
	"context"
	"encoding/json"
	"errors"
	"metaldocs/internal/platform/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/render/docgen"
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

type captureAuditWriter struct {
	events []auditdomain.Event
}

func (w *captureAuditWriter) Record(_ context.Context, event auditdomain.Event) error {
	w.events = append(w.events, event)
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

func TestCreateDocumentAssignsSequentialCodeByProfile(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	first, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-seq-1",
		Title:        "Primeiro PO",
		DocumentType: "po",
		OwnerID:      "user-1",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-001",
		},
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	second, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-seq-2",
		Title:        "Segundo PO",
		DocumentType: "po",
		OwnerID:      "user-1",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-002",
		},
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	if first.DocumentCode != "PO-001" {
		t.Fatalf("expected first code PO-001, got %s", first.DocumentCode)
	}
	if second.DocumentCode != "PO-002" {
		t.Fatalf("expected second code PO-002, got %s", second.DocumentCode)
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

func TestSaveEtapaBodyCreatesNewVersionWhenTargetIsNotDraft(t *testing.T) {
	repo := memory.NewRepository()
	pub := &capturePublisher{}
	audit := &captureAuditWriter{}
	svc := application.NewService(repo, pub, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)}).
		WithAuditWriter(audit)

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-etapa-1",
		Title:        "Procedure",
		DocumentType: "po",
		OwnerID:      "user-1",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-ETAPA-1",
		},
		InitialContent: "original-content",
		TraceID:        "trace-etapa-1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	if err := repo.UpdateDocumentStatus(context.Background(), "doc-etapa-1", domain.StatusApproved); err != nil {
		t.Fatalf("unexpected status update error: %v", err)
	}

	updated, err := svc.SaveEtapaBodyAuthorized(iamdomain.WithAuthContext(context.Background(), "editor-1", nil), domain.SaveEtapaBodyCommand{
		DocumentID:    "doc-etapa-1",
		VersionNumber: 1,
		StepIndex:     0,
		Blocks: []json.RawMessage{
			json.RawMessage(`{"type":"paragraph","text":"primeiro bloco"}`),
		},
		TraceID: "trace-etapa-2",
	})
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	if updated.Number != 2 {
		t.Fatalf("expected new version 2, got %d", updated.Number)
	}
	if len(updated.BodyBlocks) != 1 {
		t.Fatalf("expected 1 body block, got %d", len(updated.BodyBlocks))
	}
	if len(updated.BodyBlocks[0].Blocks) != 1 {
		t.Fatalf("expected 1 raw block, got %d", len(updated.BodyBlocks[0].Blocks))
	}
	if string(updated.BodyBlocks[0].Blocks[0]) != `{"type":"paragraph","text":"primeiro bloco"}` {
		t.Fatalf("unexpected block payload: %s", string(updated.BodyBlocks[0].Blocks[0]))
	}

	versions, err := svc.ListVersions(context.Background(), "doc-etapa-1")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if len(versions[0].BodyBlocks) != 0 {
		t.Fatalf("expected draft version 1 to remain unchanged, got %#v", versions[0].BodyBlocks)
	}
	if len(versions[1].BodyBlocks) != 1 {
		t.Fatalf("expected version 2 to carry body blocks, got %#v", versions[1].BodyBlocks)
	}

	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	if audit.events[0].ActorID != "editor-1" {
		t.Fatalf("expected actor editor-1, got %s", audit.events[0].ActorID)
	}
	if len(pub.events) < 3 {
		t.Fatalf("expected etapa body update event, got %d events", len(pub.events))
	}
	lastEvent := pub.events[len(pub.events)-1]
	if lastEvent.EventType != "document.etapa_body.updated" {
		t.Fatalf("expected etapa body event, got %s", lastEvent.EventType)
	}
	if lastEvent.IdempotencyKey != "etapa_body_updated:doc-etapa-1:2:0" {
		t.Fatalf("unexpected idempotency key: %s", lastEvent.IdempotencyKey)
	}
}

func TestSaveEtapaBodyRejectsInvalidBlocks(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-etapa-2",
		Title:        "Procedure",
		DocumentType: "po",
		OwnerID:      "user-2",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PO-ETAPA-2",
		},
		InitialContent: "original-content",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = svc.SaveEtapaBodyAuthorized(iamdomain.WithAuthContext(context.Background(), "editor-2", nil), domain.SaveEtapaBodyCommand{
		DocumentID:    "doc-etapa-2",
		VersionNumber: 1,
		StepIndex:     0,
		Blocks: []json.RawMessage{
			json.RawMessage("{"),
		},
	})
	if !errors.Is(err, domain.ErrInvalidNativeContent) {
		t.Fatalf("expected invalid native content error, got %v", err)
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

func TestCreateDocumentAllowsArbitraryMetadata(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-metadata",
		Title:        "Procedure With Custom Metadata",
		DocumentType: "po",
		OwnerID:      "user-1",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"wrong_field": "bad",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	aliases := map[string]string{}
	for _, item := range items {
		found[item.Code] = true
		aliases[item.Code] = item.Alias
	}

	for _, code := range []string{"po", "it", "rg"} {
		if !found[code] {
			t.Fatalf("expected profile %s in registry", code)
		}
	}
	if aliases["po"] != "Procedimentos" {
		t.Fatalf("expected alias Procedimentos for po, got %q", aliases["po"])
	}
	if aliases["it"] != "Instrucoes" {
		t.Fatalf("expected alias Instrucoes for it, got %q", aliases["it"])
	}
	if aliases["rg"] != "Registros" {
		t.Fatalf("expected alias Registros for rg, got %q", aliases["rg"])
	}
}

func TestValidateDocumentProfileAlias(t *testing.T) {
	validCases := []string{"Procedimentos", "Instrucoes", "Registros"}
	for _, alias := range validCases {
		if err := domain.ValidateDocumentProfileAlias(alias); err != nil {
			t.Fatalf("expected alias %q to be valid: %v", alias, err)
		}
	}

	invalidCases := []string{"", "   ", "Alias documental extremamente grande"}
	for _, alias := range invalidCases {
		if err := domain.ValidateDocumentProfileAlias(alias); err == nil {
			t.Fatalf("expected alias %q to be invalid", alias)
		}
	}
}

func TestValidateDocumentTypeSchema_RejectsUnknownFieldType(t *testing.T) {
	schema := domain.DocumentTypeSchema{
		Sections: []domain.SectionDef{
			{
				Key:   "s1",
				Num:   "1",
				Title: "Section 1",
				Fields: []domain.FieldDef{
					{Key: "x", Label: "X", Type: "unknown"},
				},
			},
		},
	}

	err := domain.ValidateDocumentTypeSchema(schema)
	if !errors.Is(err, domain.ErrDocumentSchemaInvalidField) {
		t.Fatalf("expected schema field error, got %v", err)
	}
	if got := err.Error(); got != "DOCUMENT_SCHEMA_INVALID_FIELD" {
		t.Fatalf("expected structured error code, got %s", got)
	}
}

func TestValidateDocumentTypeSchema_RejectsEmptySchema(t *testing.T) {
	err := domain.ValidateDocumentTypeSchema(domain.DocumentTypeSchema{})
	if !errors.Is(err, domain.ErrDocumentSchemaInvalid) {
		t.Fatalf("expected schema invalid error, got %v", err)
	}
}

func TestValidateDocumentTypeSchema_RejectsEmptySectionDefinition(t *testing.T) {
	schema := domain.DocumentTypeSchema{
		Sections: []domain.SectionDef{
			{},
		},
	}

	err := domain.ValidateDocumentTypeSchema(schema)
	if !errors.Is(err, domain.ErrDocumentSchemaInvalidSection) {
		t.Fatalf("expected section error, got %v", err)
	}
}

func TestValidateDocumentTypeSchema_RejectsEmptyTableColumns(t *testing.T) {
	schema := domain.DocumentTypeSchema{
		Sections: []domain.SectionDef{
			{
				Key:   "s1",
				Num:   "1",
				Title: "Section 1",
				Fields: []domain.FieldDef{
					{Key: "table", Label: "Table", Type: "table"},
				},
			},
		},
	}

	err := domain.ValidateDocumentTypeSchema(schema)
	if !errors.Is(err, domain.ErrDocumentSchemaInvalidField) {
		t.Fatalf("expected field error for empty table columns, got %v", err)
	}
}

func TestValidateDocumentTypeSchema_RejectsEmptyRepeatItemFields(t *testing.T) {
	schema := domain.DocumentTypeSchema{
		Sections: []domain.SectionDef{
			{
				Key:   "s1",
				Num:   "1",
				Title: "Section 1",
				Fields: []domain.FieldDef{
					{Key: "repeat", Label: "Repeat", Type: "repeat"},
				},
			},
		},
	}

	err := domain.ValidateDocumentTypeSchema(schema)
	if !errors.Is(err, domain.ErrDocumentSchemaInvalidField) {
		t.Fatalf("expected field error for empty repeat item fields, got %v", err)
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

func TestService_SaveDocumentValues_UpdatesDraftInPlace(t *testing.T) {
	repo := memory.NewRepository()
	service := application.NewService(repo, nil, nil)
	ctx := context.Background()

	doc := seedRuntimeDocument(t, repo, domain.StatusDraft)
	values := map[string]any{"objetivo": "Novo texto"}

	version, err := service.SaveDocumentValuesAuthorized(ctx, domain.SaveDocumentValuesCommand{
		DocumentID: doc.ID,
		Values:     values,
		TraceID:    "trace-runtime-save",
	})
	if err != nil {
		t.Fatalf("save values: %v", err)
	}
	if version.Number != 1 {
		t.Fatalf("expected in-place draft update, got version %d", version.Number)
	}
}

type capturedDocgenRequest struct {
	DocumentType string `json:"documentType"`
	DocumentCode string `json:"documentCode"`
	Title        string `json:"title"`
	Schema       struct {
		Sections []struct {
			Key string `json:"key"`
		} `json:"sections"`
	} `json:"schema"`
	Values map[string]any `json:"values"`
}

type runtimeDocgenStub struct {
	LastPayload capturedDocgenRequest
}

func newRuntimeServiceWithDocgenStub(t *testing.T) (*application.Service, *runtimeDocgenStub) {
	t.Helper()

	repo := memory.NewRepository()
	stub := &runtimeDocgenStub{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/generate" {
			t.Fatalf("unexpected docgen request %s %s", r.Method, r.URL.Path)
		}
		defer r.Body.Close()

		if err := json.NewDecoder(r.Body).Decode(&stub.LastPayload); err != nil {
			t.Fatalf("decode docgen payload: %v", err)
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("docx"))
	}))
	t.Cleanup(server.Close)

	service := application.NewService(repo, nil, nil).WithDocgenClient(docgen.NewClient(config.DocgenConfig{
		Enabled:               true,
		APIURL:                server.URL,
		RequestTimeoutSeconds: 5,
	}))

	doc := seedDocument("doc-1")
	doc.DocumentProfile = "po"
	doc.DocumentType = "po"
	doc.Status = domain.StatusDraft
	doc.ProfileSchemaVersion = 1
	doc.CreatedAt = time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	doc.UpdatedAt = doc.CreatedAt
	if err := repo.CreateDocument(context.Background(), doc); err != nil {
		t.Fatalf("seed runtime document: %v", err)
	}
	if err := repo.SaveVersion(context.Background(), domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "{}",
		ContentHash:   "hash-runtime-export",
		ChangeSummary: "initial runtime export",
		ContentSource: domain.ContentSourceNative,
		Values: map[string]any{
			"identification": map[string]any{
				"objetivo": "Texto de teste",
			},
		},
		CreatedAt: doc.CreatedAt,
	}); err != nil {
		t.Fatalf("seed runtime version: %v", err)
	}

	return service, stub
}

func TestService_ExportDocxUsesSchemaRuntimePayload(t *testing.T) {
	service, docgenStub := newRuntimeServiceWithDocgenStub(t)
	ctx := context.Background()

	_, err := service.ExportDocumentDocxAuthorized(ctx, "doc-1", "trace-export")
	if err != nil {
		t.Fatalf("export docx: %v", err)
	}

	if len(docgenStub.LastPayload.Schema.Sections) < 4 {
		t.Fatalf("expected runtime schema payload, got %d sections", len(docgenStub.LastPayload.Schema.Sections))
	}

	keys := make([]string, 0, len(docgenStub.LastPayload.Schema.Sections))
	for _, section := range docgenStub.LastPayload.Schema.Sections {
		keys = append(keys, section.Key)
	}
	if !containsString(keys, "identificacaoProcesso") {
		t.Fatalf("expected runtime schema sections to include identificacaoProcesso, got %v", keys)
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func seedRuntimeDocument(t *testing.T, repo *memory.Repository, status string) domain.Document {
	t.Helper()

	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	doc := seedDocument("doc-runtime")
	doc.Status = status
	doc.CreatedAt = now
	doc.UpdatedAt = now

	if err := repo.CreateDocument(context.Background(), doc); err != nil {
		t.Fatalf("seed runtime document: %v", err)
	}
	if err := repo.SaveVersion(context.Background(), domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "{}",
		ContentHash:   "hash-runtime-1",
		ChangeSummary: "initial runtime values",
		ContentSource: domain.ContentSourceNative,
		Values:        map[string]any{},
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("seed runtime version: %v", err)
	}

	return doc
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
