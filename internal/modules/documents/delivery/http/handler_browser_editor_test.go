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
	testMDDMText        = "Original"
	testMDDMTextUpdated = "Atualizado"
)

func TestHandleDocumentBrowserContentPost(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, testMDDMBody, testMDDMText)
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

	var resp DocumentContentBrowserResponse
	decodeJSONBody(t, rec.Body.Bytes(), &resp)
	if resp.DocumentID != doc.ID {
		t.Fatalf("documentID = %q, want %q", resp.DocumentID, doc.ID)
	}
	if resp.Version <= 0 {
		t.Fatalf("version = %d, want positive", resp.Version)
	}
	if resp.ContentSource != domain.ContentSourceBrowserEditor {
		t.Fatalf("contentSource = %q, want %q", resp.ContentSource, domain.ContentSourceBrowserEditor)
	}
	if resp.DraftToken == "" {
		t.Fatalf("draftToken = empty")
	}

	persisted, err := repo.GetVersion(ctx, doc.ID, resp.Version)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if persisted.Number != resp.Version {
		t.Fatalf("persisted version = %d, want %d", persisted.Number, resp.Version)
	}
	if persisted.Content != testMDDMBodyUpdated {
		t.Fatalf("persisted content = %q, want %q", persisted.Content, testMDDMBodyUpdated)
	}
	if persisted.TextContent != testMDDMTextUpdated {
		t.Fatalf("persisted textContent = %q, want %q", persisted.TextContent, testMDDMTextUpdated)
	}
	if persisted.ContentSource != domain.ContentSourceBrowserEditor {
		t.Fatalf("persisted contentSource = %q, want %q", persisted.ContentSource, domain.ContentSourceBrowserEditor)
	}
}

func TestHandleDocumentTemplatesGetAndAssignmentPut(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, testMDDMBody, testMDDMText)
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
	var listResp ListDocumentTemplatesResponse
	decodeJSONBody(t, listRec.Body.Bytes(), &listResp)
	if len(listResp.Items) == 0 {
		t.Fatalf("list items = empty")
	}
	if listResp.Items[0].TemplateKey != "po-mddm-canvas" {
		t.Fatalf("list templateKey = %q, want %q", listResp.Items[0].TemplateKey, "po-mddm-canvas")
	}
	if listResp.Items[0].Editor != "mddm-blocknote" {
		t.Fatalf("list editor = %q, want %q", listResp.Items[0].Editor, "mddm-blocknote")
	}
	if listResp.Items[0].ContentFormat != "mddm" {
		t.Fatalf("list contentFormat = %q, want %q", listResp.Items[0].ContentFormat, "mddm")
	}

	assignReq := httptest.NewRequest(http.MethodPut, "/api/v1/documents/"+doc.ID+"/template-assignment", strings.NewReader(`{"templateKey":"po-mddm-override","templateVersion":2}`))
	assignReq.Header.Set("Content-Type", "application/json")
	assignRec := httptest.NewRecorder()
	handler.handleDocumentTemplateAssignmentPut(assignRec, assignReq, doc.ID)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want %d", assignRec.Code, http.StatusOK)
	}
	var assignResp DocumentTemplateAssignmentResponse
	decodeJSONBody(t, assignRec.Body.Bytes(), &assignResp)
	if assignResp.DocumentID != doc.ID {
		t.Fatalf("assign documentID = %q, want %q", assignResp.DocumentID, doc.ID)
	}
	if assignResp.TemplateKey != "po-mddm-override" {
		t.Fatalf("assign templateKey = %q, want %q", assignResp.TemplateKey, "po-mddm-override")
	}
	if assignResp.TemplateVersion != 2 {
		t.Fatalf("assign templateVersion = %d, want %d", assignResp.TemplateVersion, 2)
	}
	if assignResp.AssignedAt != now.UTC().Format(time.RFC3339) {
		t.Fatalf("assign assignedAt = %q, want %q", assignResp.AssignedAt, now.UTC().Format(time.RFC3339))
	}
}

type applicationFixedClock struct {
	now time.Time
}

func (c applicationFixedClock) Now() time.Time {
	return c.now
}

func seedBrowserHandlerDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, body, textContent string) domain.Document {
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
		TextContent:     textContent,
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
	body := templateV2Body(t)
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, body, plainTextFromMDDMForTest(body))
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/browser-editor-bundle", nil)
	rec := httptest.NewRecorder()

	handler.handleDocumentBrowserEditorBundle(rec, req, doc.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp DocumentBrowserEditorBundleResponse
	decodeJSONBody(t, rec.Body.Bytes(), &resp)
	if resp.Document.DocumentID != doc.ID {
		t.Fatalf("documentID = %q, want %q", resp.Document.DocumentID, doc.ID)
	}
	if resp.Document.CreatedAt != now.UTC().Format(time.RFC3339) {
		t.Fatalf("createdAt = %q, want %q", resp.Document.CreatedAt, now.UTC().Format(time.RFC3339))
	}
	if len(resp.Versions) != 1 {
		t.Fatalf("versions len = %d, want %d", len(resp.Versions), 1)
	}
	if resp.Versions[0].Version != 1 {
		t.Fatalf("version = %d, want %d", resp.Versions[0].Version, 1)
	}
	if resp.Versions[0].CreatedAt != now.UTC().Format(time.RFC3339) {
		t.Fatalf("version createdAt = %q, want %q", resp.Versions[0].CreatedAt, now.UTC().Format(time.RFC3339))
	}
	if resp.TemplateSnapshot == nil {
		t.Fatalf("templateSnapshot = nil")
	}
	if resp.TemplateSnapshot.TemplateKey != "po-mddm-canvas" {
		t.Fatalf("templateSnapshot templateKey = %q, want %q", resp.TemplateSnapshot.TemplateKey, "po-mddm-canvas")
	}
	if resp.TemplateSnapshot.Editor != "mddm-blocknote" {
		t.Fatalf("templateSnapshot editor = %q, want %q", resp.TemplateSnapshot.Editor, "mddm-blocknote")
	}
	if resp.TemplateSnapshot.ContentFormat != "mddm" {
		t.Fatalf("templateSnapshot contentFormat = %q, want %q", resp.TemplateSnapshot.ContentFormat, "mddm")
	}
	assertBrowserEditorEnvelope(t, resp.Body, browserEditorSectionIDs())
	if resp.DraftToken == "" {
		t.Fatalf("draftToken = empty")
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

func decodeJSONBody(t *testing.T, data []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("decode json: %v; body = %s", err, string(data))
	}
}

func templateV2Body(t *testing.T) string {
	t.Helper()

	body, err := json.Marshal(mddm.POTemplateMDDM())
	if err != nil {
		t.Fatalf("marshal template v2 body: %v", err)
	}
	return string(body)
}

func browserEditorSectionIDs() []string {
	return []string{
		"a0000001-0000-0000-0000-000000000001",
		"a0000010-0000-0000-0000-000000000010",
		"a0000020-0000-0000-0000-000000000020",
		"a0000030-0000-0000-0000-000000000030",
		"a0000040-0000-0000-0000-000000000040",
		"a0000055-0000-0000-0000-000000000055",
		"a0000060-0000-0000-0000-000000000060",
		"a0000070-0000-0000-0000-000000000070",
		"a0000080-0000-0000-0000-000000000080",
		"a0000090-0000-0000-0000-000000000090",
	}
}

func assertBrowserEditorEnvelope(t *testing.T, body string, expectedSectionIDs []string) map[string]any {
	t.Helper()

	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("bundle body is not valid JSON: %v", err)
	}

	if got, ok := envelope["mddm_version"].(float64); !ok || got != 1 {
		t.Fatalf("bundle body mddm_version = %#v, want 1", envelope["mddm_version"])
	}
	if _, ok := envelope["template_ref"]; !ok {
		t.Fatal("bundle body is missing template_ref")
	}

	blocks, ok := envelope["blocks"].([]any)
	if !ok {
		t.Fatalf("bundle body blocks = %#v, want array", envelope["blocks"])
	}
	if len(blocks) != len(expectedSectionIDs) {
		t.Fatalf("bundle body section count = %d, want %d", len(blocks), len(expectedSectionIDs))
	}

	for i, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]any)
		if !ok {
			t.Fatalf("bundle body blocks[%d] = %#v, want object", i, rawBlock)
		}
		if got := block["type"]; got != "section" {
			t.Fatalf("bundle body blocks[%d].type = %#v, want section", i, got)
		}
		if got := block["id"]; got != expectedSectionIDs[i] {
			t.Fatalf("bundle body section[%d].id = %#v, want %q", i, got, expectedSectionIDs[i])
		}
		props, ok := block["props"].(map[string]any)
		if !ok {
			t.Fatalf("bundle body blocks[%d].props = %#v, want object", i, block["props"])
		}
		title, ok := props["title"].(string)
		if !ok || title == "" {
			t.Fatalf("bundle body blocks[%d].props.title = %#v, want non-empty string", i, props["title"])
		}
		children, ok := block["children"].([]any)
		if !ok {
			t.Fatalf("bundle body blocks[%d].children = %#v, want array", i, block["children"])
		}
		if len(children) == 0 {
			t.Fatalf("bundle body blocks[%d] has no children", i)
		}
	}

	return envelope
}

func plainTextFromMDDMForTest(body string) string {
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

	parts := make([]string, 0)
	for _, block := range envelope.Blocks {
		collectMDDMTextForTest(block, &parts)
	}
	return strings.Join(parts, " ")
}

func collectMDDMTextForTest(raw json.RawMessage, parts *[]string) {
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
		collectMDDMTextForTest(child, parts)
	}
}
