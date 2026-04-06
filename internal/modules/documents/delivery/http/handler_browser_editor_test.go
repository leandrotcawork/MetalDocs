package httpdelivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestHandleDocumentBrowserContentPost(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+doc.ID+"/content/browser", strings.NewReader(`{"body":"<p>Atualizado</p>","draftToken":"v1:test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleDocumentContentBrowserPost(rec, req, doc.ID)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if !strings.Contains(rec.Body.String(), `"contentSource":"browser_editor"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHandleDocumentTemplatesGetAndAssignmentPut(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-browser-override",
		Version:       2,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Browser Override",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><p>Override</p></section>`,
		CreatedAt:     time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}
	handler := NewHandler(service)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/document-templates?profileCode=po", nil)
	listRec := httptest.NewRecorder()
	handler.handleDocumentTemplates(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	if !strings.Contains(listRec.Body.String(), `"templateKey":"po-default-browser"`) {
		t.Fatalf("list body = %s", listRec.Body.String())
	}

	assignReq := httptest.NewRequest(http.MethodPut, "/api/v1/documents/"+doc.ID+"/template-assignment", strings.NewReader(`{"templateKey":"po-browser-override","templateVersion":2}`))
	assignReq.Header.Set("Content-Type", "application/json")
	assignRec := httptest.NewRecorder()
	handler.handleDocumentTemplateAssignmentPut(assignRec, assignReq, doc.ID)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want %d", assignRec.Code, http.StatusOK)
	}
	if !strings.Contains(assignRec.Body.String(), `"templateKey":"po-browser-override"`) {
		t.Fatalf("assign body = %s", assignRec.Body.String())
	}
	if !strings.Contains(assignRec.Body.String(), `"templateVersion":2`) {
		t.Fatalf("assign body = %s", assignRec.Body.String())
	}
}

type applicationFixedClock struct {
	now time.Time
}

func (c applicationFixedClock) Now() time.Time {
	return c.now
}

func seedBrowserHandlerDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, body string) domain.Document {
	t.Helper()

	seedCompatibleBrowserTemplateSchemaSet(t, repo)

	doc := domain.Document{
		ID:                   "doc-123",
		Title:                "Browser Handler Document",
		DocumentType:         "po",
		DocumentProfile:      "po",
		DocumentFamily:       "procedure",
		DocumentSequence:     123,
		DocumentCode:         "PO-123",
		ProfileSchemaVersion: 1,
		OwnerID:              "owner-1",
		BusinessUnit:         "operations",
		Department:           "sgq",
		Classification:       domain.ClassificationInternal,
		Status:               domain.StatusDraft,
		Tags:                 []string{},
		MetadataJSON:         map[string]any{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := repo.CreateDocument(ctx, doc); err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := repo.UpsertDocumentTemplateAssignment(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      doc.ID,
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
		AssignedAt:      now,
	}); err != nil {
		t.Fatalf("upsert template assignment: %v", err)
	}
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:      doc.ID,
		Number:          1,
		Content:         body,
		ContentHash:     "test",
		ChangeSummary:   "Initial browser draft",
		ContentSource:   domain.ContentSourceBrowserEditor,
		TextContent:     "Original",
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	return doc
}

func seedCompatibleBrowserTemplateSchemaSet(t *testing.T, repo *documentmemory.Repository) {
	t.Helper()

	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
		ProfileCode:   "po",
		Version:       3,
		IsActive:      false,
		ContentSchema: map[string]any{},
	}); err != nil {
		t.Fatalf("upsert browser schema version: %v", err)
	}
}
