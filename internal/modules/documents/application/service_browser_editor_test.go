package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestGetBrowserEditorBundleReturnsDraftHTML(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocument(t, ctx, repo, now, body)

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}
	assertBrowserEditorEnvelope(t, bundle.Body, browserEditorSectionTitles())
	if bundle.DraftToken == "" {
		t.Fatal("expected draft token")
	}
	if bundle.TemplateSnapshot.TemplateKey != "po-default-canvas" {
		t.Fatalf("template key = %q, want po-default-canvas", bundle.TemplateSnapshot.TemplateKey)
	}
	if !bundle.TemplateSnapshot.IsBrowserHTML() {
		t.Fatalf("template snapshot = %#v, want browser html", bundle.TemplateSnapshot)
	}
}

func TestGetBrowserEditorBundleRequiresTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocumentWithoutTemplate(t, ctx, repo, now, body)

	_, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if !errors.Is(err, domain.ErrDocumentTemplateNotFound) {
		t.Fatalf("err = %v, want ErrDocumentTemplateNotFound", err)
	}
}

func TestSaveBrowserContentAuthorizedUpdatesDraftInPlace(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocument(t, ctx, repo, now, body)
	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	version, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if err != nil {
		t.Fatalf("SaveBrowserContentAuthorized() error = %v", err)
	}
	if version.Number != 1 {
		t.Fatalf("version number = %d, want 1", version.Number)
	}
	if version.ContentSource != domain.ContentSourceBrowserEditor {
		t.Fatalf("content source = %q, want %q", version.ContentSource, domain.ContentSourceBrowserEditor)
	}
	if version.Content != `<section><p>Atualizado</p></section>` {
		t.Fatalf("content = %q, want updated HTML", version.Content)
	}

	savedVersion, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() after save error = %v", err)
	}
	if savedVersion.Content != `<section><p>Atualizado</p></section>` {
		t.Fatalf("saved content = %q, want updated HTML", savedVersion.Content)
	}
}

func TestSaveBrowserContentAuthorizedRequiresTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocumentWithoutTemplate(t, ctx, repo, now, body)
	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	_, err = service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if !errors.Is(err, domain.ErrDocumentTemplateNotFound) {
		t.Fatalf("err = %v, want ErrDocumentTemplateNotFound", err)
	}
}

func TestSaveBrowserContentAuthorizedRejectsNonBrowserTemplate(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocument(t, ctx, repo, now, body)

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   "po-governed-docx",
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO Governed DOCX",
		Editor:        "docx",
		ContentFormat: "json",
		Definition: map[string]any{
			"type": "page",
			"id":   "po-root",
			"children": []any{
				map[string]any{
					"type":  "section-frame",
					"id":    "section-visao-geral",
					"title": "Visao Geral",
					"children": []any{
						map[string]any{"type": "rich-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "rich"},
					},
				},
			},
		},
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertDocumentTemplateVersionForTest() error = %v", err)
	}

	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	current.TemplateKey = "po-governed-docx"
	current.TemplateVersion = 1
	if err := repo.UpdateDraftVersionContentCAS(ctx, current, current.ContentHash); err != nil {
		t.Fatalf("UpdateDraftVersionContentCAS() error = %v", err)
	}

	_, err = service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestGetBrowserEditorBundleRejectsIncompatibleStoredTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocument(t, ctx, repo, now, body)

	setStoredBrowserTemplateSnapshotForTest(t, ctx, repo, now, doc.ID, "po-browser-invalid-schema", 99)

	_, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestSaveBrowserContentAuthorizedRejectsIncompatibleStoredTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocument(t, ctx, repo, now, body)

	current := setStoredBrowserTemplateSnapshotForTest(t, ctx, repo, now, doc.ID, "po-browser-invalid-schema", 99)

	_, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       `<section><p>Atualizado</p></section>`,
		TraceID:    "trace-test",
	})
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want ErrInvalidCommand", err)
	}
}

func seedBrowserDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, body string) domain.Document {
	t.Helper()

	seedCompatiblePOProfileSchemaSet(t, repo)

	doc := seedDraftDocument(t, ctx, repo, now)
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
		ContentHash:     contentHash(body),
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

func setStoredBrowserTemplateSnapshotForTest(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, documentID, templateKey string, schemaVersion int) domain.Version {
	t.Helper()

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   templateKey,
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: schemaVersion,
		Name:          "PO Browser Invalid Snapshot",
		Editor:        "ckeditor5",
		ContentFormat: "html",
		Body:          `<section><p>Stored invalid browser snapshot</p></section>`,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("UpsertDocumentTemplateVersionForTest() error = %v", err)
	}

	current, err := repo.GetVersion(ctx, documentID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	current.TemplateKey = templateKey
	current.TemplateVersion = 1
	if err := repo.UpdateDraftVersionContentCAS(ctx, current, current.ContentHash); err != nil {
		t.Fatalf("UpdateDraftVersionContentCAS() error = %v", err)
	}
	return current
}

func seedBrowserDocumentWithoutTemplate(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, body string) domain.Document {
	t.Helper()

	seedCompatiblePOProfileSchemaSet(t, repo)
	doc := seedDraftDocument(t, ctx, repo, now)
	if err := repo.UpsertDocumentTemplateAssignment(ctx, domain.DocumentTemplateAssignment{
		DocumentID:      doc.ID,
		TemplateKey:     "po-missing-template",
		TemplateVersion: 99,
		AssignedAt:      now,
	}); err != nil {
		t.Fatalf("upsert template assignment: %v", err)
	}

	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       body,
		ContentHash:   contentHash(body),
		ChangeSummary: "Initial browser draft",
		ContentSource: domain.ContentSourceBrowserEditor,
		TextContent:   browserEditorBodyText(t, body),
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}
	return doc
}

func TestNewPODocumentGetsBrowserTemplateInBundle(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})

	seedCompatiblePOProfileSchemaSet(t, repo)

	doc, err := service.CreateDocument(ctx, domain.CreateDocumentCommand{
		DocumentID:      "doc-browser-smoke",
		Title:           "PO Smoke Test",
		DocumentType:    "po",
		DocumentProfile: "po",
		OwnerID:         "owner-1",
		BusinessUnit:    "operations",
		Department:      "sgq",
		InitialContent:  `{"legacy":"content"}`,
		TraceID:         "trace-browser-smoke",
	})
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}

	if bundle.TemplateSnapshot.TemplateKey != "po-default-browser" {
		t.Fatalf("template key = %q, want po-default-browser", bundle.TemplateSnapshot.TemplateKey)
	}
	if bundle.TemplateSnapshot.Version != 1 {
		t.Fatalf("template version = %d, want 1", bundle.TemplateSnapshot.Version)
	}
	if !bundle.TemplateSnapshot.IsBrowserHTML() {
		t.Fatalf("template snapshot = %#v, want browser html", bundle.TemplateSnapshot)
	}
	if bundle.Body == "" {
		t.Fatal("bundle body is empty")
	}
	// Title is rendered by DocumentEditorHeader React component; body starts with the first section.
	if !strings.Contains(bundle.Body, "Identificação do Processo") {
		t.Fatal("bundle body does not contain template content")
	}
	if bundle.DraftToken == "" {
		t.Fatal("expected draft token")
	}
}

func TestSubstituteTemplateTokens(t *testing.T) {
	doc := domain.Document{
		OwnerID:   "leandro_theodoro",
		CreatedAt: time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
	}
	version := domain.Version{Number: 1}

	body := `<td><p class="restricted-editing-exception">{{versao}}</p></td>` +
		`<td><p class="restricted-editing-exception">{{data_criacao}}</p></td>` +
		`<td><p class="restricted-editing-exception"></p></td>` +
		`<td><p class="restricted-editing-exception">{{elaborador}}</p></td>`

	got := substituteTemplateTokens(body, doc, version)

	if strings.Contains(got, "{{versao}}") {
		t.Error("expected {{versao}} to be replaced")
	}
	if strings.Contains(got, "{{data_criacao}}") {
		t.Error("expected {{data_criacao}} to be replaced")
	}
	if strings.Contains(got, "{{elaborador}}") {
		t.Error("expected {{elaborador}} to be replaced")
	}
	if !strings.Contains(got, "01") {
		t.Error("expected version 01 in result")
	}
	if !strings.Contains(got, "06/04/2026") {
		t.Error("expected date 06/04/2026 in result")
	}
	if !strings.Contains(got, "leandro_theodoro") {
		t.Error("expected owner in result")
	}
}

func TestSubstituteTemplateTokensIdempotent(t *testing.T) {
	doc := domain.Document{
		OwnerID:   "owner",
		CreatedAt: time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
	}
	version := domain.Version{Number: 1}

	// Body with no tokens — should be returned unchanged
	body := `<p class="restricted-editing-exception">already filled</p>`
	got := substituteTemplateTokens(body, doc, version)
	if got != body {
		t.Errorf("idempotent: body without tokens should be unchanged, got %q", got)
	}
}

func TestSubstituteTemplateTokensEmptyOwner(t *testing.T) {
	doc := domain.Document{} // zero value: no owner, zero time
	version := domain.Version{Number: 2}

	body := `{{versao}} {{data_criacao}} {{elaborador}}`
	got := substituteTemplateTokens(body, doc, version)

	if !strings.Contains(got, "02") {
		t.Error("expected version 02")
	}
	if !strings.Contains(got, "—") {
		t.Error("expected em dash fallback for missing owner and date")
	}
}

// TestGetBrowserEditorBundleSubstitutesTokens exercises GetBrowserEditorBundleAuthorized
// end-to-end and asserts no raw tokens remain in bundle.Body.
// Note: After Task 6 rewrites the PO browser template body to include
// {{versao}}, {{data_criacao}}, {{elaborador}} tokens, this test will also
// validate that the substitution pipeline runs on those tokens. Currently it
// guards against accidental raw-token leakage in the bundle path.
func TestGetBrowserEditorBundleSubstitutesTokens(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	body := templateV2Body(t)
	doc := seedBrowserDocument(t, ctx, repo, now, body)

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}

	envelope := assertBrowserEditorEnvelope(t, bundle.Body, browserEditorSectionTitles())
	assertBrowserEditorBodyHasNoTokens(t, envelope, []string{"{{versao}}", "{{data_criacao}}", "{{elaborador}}"})
	if bundle.Body == "" {
		t.Fatal("bundle body is empty")
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

func assertBrowserEditorBodyHasNoTokens(t *testing.T, value any, tokens []string) {
	t.Helper()
	assertBrowserEditorBodyHasNoTokensAtPath(t, value, "body", tokens)
}

func assertBrowserEditorBodyHasNoTokensAtPath(t *testing.T, value any, path string, tokens []string) {
	t.Helper()

	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			assertBrowserEditorBodyHasNoTokensAtPath(t, child, fmt.Sprintf("%s.%s", path, key), tokens)
		}
	case []any:
		for i, child := range v {
			assertBrowserEditorBodyHasNoTokensAtPath(t, child, fmt.Sprintf("%s[%d]", path, i), tokens)
		}
	case string:
		for _, token := range tokens {
			if strings.Contains(v, token) {
				t.Fatalf("%s contains raw token %q", path, token)
			}
		}
	}
}
