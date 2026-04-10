package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/domain/mddm"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

const (
	testMDDMBody        = `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"Original"}]}]}`
	testMDDMBodyUpdated = `{"mddm_version":1,"template_ref":null,"blocks":[{"id":"b1","type":"paragraph","props":{},"children":[{"text":"Atualizado"}]}]}`
)

func TestGetBrowserEditorBundleReturnsDraftMDDM(t *testing.T) {
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
	if bundle.DraftToken == "" {
		t.Fatal("expected draft token")
	}
	if bundle.TemplateSnapshot.TemplateKey != "po-mddm-canvas" {
		t.Fatalf("template key = %q, want po-mddm-canvas", bundle.TemplateSnapshot.TemplateKey)
	}
	if bundle.TemplateSnapshot.Editor != "mddm-blocknote" {
		t.Fatalf("template editor = %q, want mddm-blocknote", bundle.TemplateSnapshot.Editor)
	}
	if bundle.TemplateSnapshot.ContentFormat != "mddm" {
		t.Fatalf("template contentFormat = %q, want mddm", bundle.TemplateSnapshot.ContentFormat)
	}
	assertBrowserEditorEnvelope(t, bundle.Body, browserEditorSectionIDs())
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
	doc := seedBrowserDocumentWithoutTemplate(t, ctx, repo, now, testMDDMBody)

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
	doc := seedBrowserDocument(t, ctx, repo, now, testMDDMBody)
	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	version, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       testMDDMBodyUpdated,
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
	if version.Content != testMDDMBodyUpdated {
		t.Fatalf("content = %q, want updated MDDM body", version.Content)
	}
	if version.TextContent != "Atualizado" {
		t.Fatalf("version text content = %q, want extracted MDDM text", version.TextContent)
	}

	savedVersion, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() after save error = %v", err)
	}
	if savedVersion.Content != testMDDMBodyUpdated {
		t.Fatalf("saved content = %q, want updated MDDM body", savedVersion.Content)
	}
	if savedVersion.TextContent != "Atualizado" {
		t.Fatalf("saved text content = %q, want extracted MDDM text", savedVersion.TextContent)
	}
}

func TestSaveBrowserContentAuthorizedRequiresTemplateSnapshot(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 4, 11, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedBrowserDocumentWithoutTemplate(t, ctx, repo, now, testMDDMBody)
	current, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}

	_, err = service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       testMDDMBodyUpdated,
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
	doc := seedBrowserDocument(t, ctx, repo, now, testMDDMBody)

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
		Body:       testMDDMBodyUpdated,
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
	doc := seedBrowserDocument(t, ctx, repo, now, testMDDMBody)

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
	doc := seedBrowserDocument(t, ctx, repo, now, testMDDMBody)

	current := setStoredBrowserTemplateSnapshotForTest(t, ctx, repo, now, doc.ID, "po-browser-invalid-schema", 99)

	_, err := service.SaveBrowserContentAuthorized(ctx, domain.SaveBrowserContentCommand{
		DocumentID: doc.ID,
		DraftToken: draftTokenForVersion(current),
		Body:       testMDDMBodyUpdated,
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
		ContentHash:     contentHash(body),
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

func setStoredBrowserTemplateSnapshotForTest(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time, documentID, templateKey string, schemaVersion int) domain.Version {
	t.Helper()

	if err := repo.UpsertDocumentTemplateVersionForTest(ctx, domain.DocumentTemplateVersion{
		TemplateKey:   templateKey,
		Version:       1,
		ProfileCode:   "po",
		SchemaVersion: schemaVersion,
		Name:          "PO Browser Invalid Snapshot",
		Editor:        "mddm-blocknote",
		ContentFormat: "mddm",
		Body:          testMDDMBody,
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
		InitialContent:  "",
		TraceID:         "trace-browser-smoke",
	})
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}

	bundle, err := service.GetBrowserEditorBundleAuthorized(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetBrowserEditorBundleAuthorized() error = %v", err)
	}

	if bundle.TemplateSnapshot.TemplateKey != "po-mddm-canvas" {
		t.Fatalf("template key = %q, want po-mddm-canvas", bundle.TemplateSnapshot.TemplateKey)
	}
	if bundle.TemplateSnapshot.Version != 1 {
		t.Fatalf("template version = %d, want 1", bundle.TemplateSnapshot.Version)
	}
	if bundle.TemplateSnapshot.Editor != "mddm-blocknote" || bundle.TemplateSnapshot.ContentFormat != "mddm" {
		t.Fatalf("template snapshot = %#v, want mddm-blocknote/mddm", bundle.TemplateSnapshot)
	}
	if bundle.Body != "" {
		t.Fatalf("bundle body = %q, want empty string by design for new MDDM draft", bundle.Body)
	}
	if bundle.DraftToken == "" {
		t.Fatal("expected draft token")
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
