package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/domain/mddm"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

const (
	testMDDMBody        = `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"Original"}]}]}`
	testMDDMBodyUpdated = `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"Atualizado"}]}]}`
)

func TestHandleDocumentBrowserContentPost(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, testMDDMBody)
	handler := NewHandler(service)

	reqBody, err := json.Marshal(map[string]string{
		"body":       testMDDMBodyUpdated,
		"draftToken": "v1:test",
	})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+doc.ID+"/content/browser", bytes.NewReader(reqBody))
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
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, testMDDMBody)
	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-mddm-override",
		Version:       2,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO MDDM Override",
		Editor:        "mddm-blocknote",
		ContentFormat: "mddm",
		Body:          testMDDMBodyUpdated,
		Definition:    mddm.POTemplateMDDM(),
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
	if !strings.Contains(listRec.Body.String(), `"templateKey":"po-mddm-canvas"`) {
		t.Fatalf("list body = %s", listRec.Body.String())
	}

	assignReq := httptest.NewRequest(http.MethodPut, "/api/v1/documents/"+doc.ID+"/template-assignment", strings.NewReader(`{"templateKey":"po-mddm-override","templateVersion":2}`))
	assignReq.Header.Set("Content-Type", "application/json")
	assignRec := httptest.NewRecorder()
	handler.handleDocumentTemplateAssignmentPut(assignRec, assignReq, doc.ID)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want %d", assignRec.Code, http.StatusOK)
	}
	if !strings.Contains(assignRec.Body.String(), `"templateKey":"po-mddm-override"`) {
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
		TemplateKey:     "po-mddm-canvas",
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
		TextContent:     plainTextFromMDDM(body),
		TemplateKey:     "po-mddm-canvas",
		TemplateVersion: 1,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	return doc
}

func TestHandleDocumentBrowserEditorBundleCreatedAt(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/browser-editor-bundle", nil)
	rec := httptest.NewRecorder()

	handler.handleDocumentBrowserEditorBundle(rec, req, doc.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"createdAt":"`) {
		t.Fatalf("expected createdAt in response, got: %s", body)
	}
	wantCreatedAt := now.UTC().Format(time.RFC3339)
	if !strings.Contains(body, `"createdAt":"`+wantCreatedAt+`"`) {
		t.Fatalf("expected createdAt %q in response, got: %s", wantCreatedAt, body)
	}
}

func seedCompatibleBrowserTemplateSchemaSet(t *testing.T, repo *documentmemory.Repository) {
	t.Helper()

	schema := map[string]any{
		"sections": []any{
			map[string]any{
				"key":   "identificacaoProcesso",
				"num":   "2",
				"title": "Identificacao do Processo",
				"fields": []any{
					map[string]any{"key": "objetivo", "label": "Objetivo", "type": "textarea"},
				},
			},
			map[string]any{
				"key":   "visaoGeral",
				"num":   "4",
				"title": "Visao Geral do Processo",
				"fields": []any{
					map[string]any{"key": "descricaoProcesso", "label": "Descricao do processo", "type": "rich"},
				},
			},
		},
	}

	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
		ProfileCode:   "po",
		Version:       1,
		IsActive:      true,
		ContentSchema: schema,
	}); err != nil {
		t.Fatalf("upsert browser schema version: %v", err)
	}
	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
		ProfileCode:   "po",
		Version:       3,
		IsActive:      false,
		ContentSchema: schema,
	}); err != nil {
		t.Fatalf("upsert browser schema version: %v", err)
	}
	if err := repo.UpsertDocumentTemplateVersionForTest(context.Background(), domain.DocumentTemplateVersion{
		TemplateKey:   "po-mddm-canvas",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO MDDM Canvas v1",
		Editor:        "mddm-blocknote",
		ContentFormat: "mddm",
		Body:          testMDDMBody,
		Definition:    mddm.POTemplateMDDM(),
		CreatedAt:     time.Unix(0, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert browser template version: %v", err)
	}
}

func plainTextFromMDDM(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	var envelope struct {
		Blocks []json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return ""
	}

	var parts []string
	for _, block := range envelope.Blocks {
		collectPlainTextFromMDDM(block, &parts)
	}
	return strings.Join(parts, " ")
}

func collectPlainTextFromMDDM(raw json.RawMessage, parts *[]string) {
	var node struct {
		Text     string            `json:"text"`
		Children []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(raw, &node); err != nil {
		return
	}
	if text := strings.TrimSpace(node.Text); text != "" {
		*parts = append(*parts, text)
	}
	for _, child := range node.Children {
		collectPlainTextFromMDDM(child, parts)
	}
}
