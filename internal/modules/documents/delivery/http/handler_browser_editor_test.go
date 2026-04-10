package httpdelivery

import (
	"context"
	"encoding/json"
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
	body := templateV2Body(t)
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, body)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/"+doc.ID+"/content/browser", strings.NewReader(`{"body":"<p>Atualizado</p>","draftToken":"v1:test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleDocumentContentBrowserPost(rec, req, doc.ID)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	var response DocumentContentBrowserResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal browser content response: %v", err)
	}
	if response.ContentSource != "browser_editor" {
		t.Fatalf("contentSource = %q, want browser_editor", response.ContentSource)
	}
}

func TestHandleDocumentTemplatesGetAndAssignmentPut(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := application.NewService(repo, nil, applicationFixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, body)
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
		TextContent:     browserEditorBodyText(t, body),
		TemplateKey:     "po-default-canvas",
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
	doc := seedBrowserHandlerDocument(t, ctx, repo, now, body)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/browser-editor-bundle", nil)
	rec := httptest.NewRecorder()

	handler.handleDocumentBrowserEditorBundle(rec, req, doc.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response DocumentBrowserEditorBundleResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal browser editor bundle response: %v", err)
	}
	wantCreatedAt := now.UTC().Format(time.RFC3339)
	if response.Document.CreatedAt != wantCreatedAt {
		t.Fatalf("createdAt = %q, want %q", response.Document.CreatedAt, wantCreatedAt)
	}
	assertBrowserEditorEnvelope(t, response.Body, browserEditorSectionTitles())
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

func browserEditorSectionTitles() []string {
	return []string{
		"Identificação do Processo",
		"Entradas e Saídas",
		"Visão Geral do Processo",
		"Detalhamento das Etapas",
		"Controle e Exceções",
		"Indicadores de Desempenho",
		"Documentos e Referências",
		"Glossário",
		"Histórico de Revisões",
	}
}

func templateV2Body(t *testing.T) string {
	t.Helper()

	body := map[string]any{
		"mddm_version": 1,
		"template_ref": nil,
		"blocks": []any{
			browserEditorSectionBlock("section-identificacao-processo", "Identificação do Processo",
				browserEditorParagraphBlock("identificacao-objetivo", "Objetivo"),
				browserEditorParagraphBlock("identificacao-escopo", "Escopo"),
				browserEditorParagraphBlock("identificacao-cargo", "Cargo responsável"),
				browserEditorParagraphBlock("identificacao-canal", "Canal / Contexto"),
				browserEditorParagraphBlock("identificacao-participantes", "Participantes"),
			),
			browserEditorSectionBlock("section-entradas-saidas", "Entradas e Saídas",
				browserEditorParagraphBlock("entradas", "Entradas"),
				browserEditorParagraphBlock("saidas", "Saídas"),
				browserEditorParagraphBlock("documentos-relacionados", "Documentos relacionados"),
				browserEditorParagraphBlock("sistemas-utilizados", "Sistemas utilizados"),
			),
			browserEditorSectionBlock("section-visao-geral", "Visão Geral do Processo",
				browserEditorParagraphBlock("descricao-processo", "Descrição do processo"),
				browserEditorParagraphBlock("ferramenta-fluxograma", "Ferramenta do fluxograma"),
				browserEditorParagraphBlock("link-fluxograma", "Link do fluxograma"),
				browserEditorParagraphBlock("diagrama", "Diagrama"),
			),
			browserEditorSectionBlock("section-detalhamento-etapas", "Detalhamento das Etapas",
				browserEditorParagraphBlock("etapa-1", "Etapa 1 - [Nome da etapa]"),
				browserEditorParagraphBlock("etapa-1-descricao", "Descreva esta etapa livremente."),
			),
			browserEditorSectionBlock("section-controle-excecoes", "Controle e Exceções",
				browserEditorParagraphBlock("pontos-controle", "Pontos de controle"),
				browserEditorParagraphBlock("excecoes-desvios", "Exceções e desvios"),
			),
			browserEditorSectionBlock("section-indicadores-desempenho", "Indicadores de Desempenho",
				browserEditorParagraphBlock("kpis", "KPIs"),
			),
			browserEditorSectionBlock("section-documentos-referencias", "Documentos e Referências",
				browserEditorParagraphBlock("documentos-referencias-titulo", "Documentos e referências"),
			),
			browserEditorSectionBlock("section-glossario", "Glossário",
				browserEditorParagraphBlock("glossario-titulo", "Glossário"),
			),
			browserEditorSectionBlock("section-historico-revisoes", "Histórico de Revisões",
				browserEditorParagraphBlock("historico-versao", "{{versao}}"),
				browserEditorParagraphBlock("historico-data", "{{data_criacao}}"),
				browserEditorParagraphBlock("historico-autoria", "{{elaborador}}"),
			),
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal template body: %v", err)
	}
	return string(raw)
}

func browserEditorSectionBlock(id, title string, children ...any) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "section",
		"props": map[string]any{
			"title":  title,
			"color":  "#6b1f2a",
			"locked": false,
		},
		"children": children,
	}
}

func browserEditorParagraphBlock(id, text string) map[string]any {
	return map[string]any{
		"id":   id,
		"type": "paragraph",
		"props": map[string]any{
			"locked": false,
		},
		"children": []any{
			map[string]any{
				"text": text,
			},
		},
	}
}

func browserEditorBodyText(t *testing.T, body string) string {
	t.Helper()

	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("unmarshal browser body: %v", err)
	}

	parts := make([]string, 0, 32)
	collectBrowserEditorText(&parts, parsed)
	return strings.Join(parts, " ")
}

func collectBrowserEditorText(parts *[]string, value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "title" || key == "text" {
				if text, ok := child.(string); ok && strings.TrimSpace(text) != "" {
					*parts = append(*parts, text)
				}
			}
			collectBrowserEditorText(parts, child)
		}
	case []any:
		for _, child := range v {
			collectBrowserEditorText(parts, child)
		}
	}
}

func assertBrowserEditorEnvelope(t *testing.T, body string, expectedSectionTitles []string) map[string]any {
	t.Helper()

	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("bundle body is not valid JSON: %v", err)
	}

	if _, ok := envelope["mddm_version"]; !ok {
		t.Fatal("bundle body is missing mddm_version")
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
	if len(blocks) != len(expectedSectionTitles) {
		t.Fatalf("bundle body section count = %d, want %d", len(blocks), len(expectedSectionTitles))
	}

	for i, rawBlock := range blocks {
		block, ok := rawBlock.(map[string]any)
		if !ok {
			t.Fatalf("bundle body blocks[%d] = %#v, want object", i, rawBlock)
		}
		if got := block["type"]; got != "section" {
			t.Fatalf("bundle body blocks[%d].type = %#v, want section", i, got)
		}
		props, ok := block["props"].(map[string]any)
		if !ok {
			t.Fatalf("bundle body blocks[%d].props = %#v, want object", i, block["props"])
		}
		title, ok := props["title"].(string)
		if !ok {
			t.Fatalf("bundle body blocks[%d].props.title = %#v, want string", i, props["title"])
		}
		if title != expectedSectionTitles[i] {
			t.Fatalf("bundle body section[%d] title = %q, want %q", i, title, expectedSectionTitles[i])
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
