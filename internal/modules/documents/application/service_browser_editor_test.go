package application

import (
	"context"
	"errors"
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
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}
	if bundle.Body != `<section><p>Original</p></section>` {
		t.Fatalf("bundle body = %q, want original HTML", bundle.Body)
	}
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

func TestPlainTextFromMDDM(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "empty body",
			body: "",
			want: "",
		},
		{
			name: "invalid json",
			body: "not json",
			want: "",
		},
		{
			name: "empty blocks",
			body: `{"mddm_version":1,"template_ref":null,"blocks":[]}`,
			want: "",
		},
		{
			name: "single paragraph",
			body: `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"Hello world"}]}]}`,
			want: "Hello world",
		},
		{
			name: "multiple paragraphs",
			body: `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"First"}]},{"id":"b2","type":"paragraph","props":{},"children":[{"text":"Second"}]}]}`,
			want: "First Second",
		},
		{
			name: "nested section field",
			body: `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"s1","type":"section","props":{"title":"Section"},"children":[{"id":"f1","type":"field","props":{"label":"Field","valueMode":"inline"},"children":[{"text":"Nested value"}]}]}]}`,
			want: "Nested value",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := plainTextFromMDDM(tc.body)
			if got != tc.want {
				t.Fatalf("plainTextFromMDDM(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestGetBrowserEditorBundleRequiresTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocumentWithoutTemplate(t, ctx, repo, now, `<section><p>Original</p></section>`)

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
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)
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
	doc := seedBrowserDocumentWithoutTemplate(t, ctx, repo, now, `<section><p>Original</p></section>`)
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
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

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
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

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
	doc := seedBrowserDocument(t, ctx, repo, now, `<section><p>Original</p></section>`)

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
		TextContent:     plainTextFromMDDM(body),
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
		TextContent:   plainTextFromMDDM(body),
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

	seedCompatiblePOProfileSchemaSet(t, repo)

	doc, err := service.CreateDocument(ctx, domain.CreateDocumentCommand{
		DocumentID:      "doc-token-substitution",
		Title:           "Token Substitution Test",
		DocumentType:    "po",
		DocumentProfile: "po",
		OwnerID:         "leandro_theodoro",
		BusinessUnit:    "operations",
		Department:      "sgq",
		InitialContent:  `{"legacy":"content"}`,
		TraceID:         "trace-token-test",
	})
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}

	// Assert: no raw tokens remain in bundle.Body (tokens present after Task 6 rewrite)
	for _, token := range []string{"{{versao}}", "{{data_criacao}}", "{{elaborador}}"} {
		if strings.Contains(bundle.Body, token) {
			t.Errorf("bundle.Body must not contain raw token %q after substitution", token)
		}
	}
	// Assert: bundle has content (template body is present)
	if bundle.Body == "" {
		t.Fatal("bundle body is empty")
	}
}
