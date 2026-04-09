package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/render/docgen"
	"metaldocs/internal/platform/render/gotenberg"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestRenderContentPDFAuthorizedCachesGeneratedDocxKey(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	store := documentmemory.NewAttachmentStore()

	doc := seedDraftDocument(t, ctx, repo, now)
	seedCompatiblePOProfileSchemaSet(t, repo)
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "{}",
		ContentHash:   contentHash("{}"),
		ChangeSummary: "Initial version",
		ContentSource: domain.ContentSourceNative,
		NativeContent: map[string]any{},
		TextContent:   "{}",
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	docgenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate" {
			t.Fatalf("docgen path = %q, want /generate", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("docx-binary"))
	}))
	defer docgenServer.Close()

	gotenbergServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forms/libreoffice/convert" {
			t.Fatalf("gotenberg path = %q, want /forms/libreoffice/convert", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer gotenbergServer.Close()

	service := NewService(repo, nil, fixedClock{now: now}).
		WithAttachmentStore(store).
		WithDocgenClient(docgen.NewClient(config.DocgenConfig{
			Enabled:               true,
			APIURL:                docgenServer.URL,
			RequestTimeoutSeconds: 1,
		})).
		WithGotenberg(gotenberg.NewClient(gotenbergServer.URL))

	version, err := service.RenderContentPDFAuthorized(ctx, doc.ID, "trace-render")
	if err != nil {
		t.Fatalf("render content pdf: %v", err)
	}

	docxKey := documentContentStorageKey(doc.ID, 1, "docx")
	if _, err := store.Open(ctx, docxKey); err != nil {
		t.Fatalf("open cached docx %q: %v", docxKey, err)
	}

	savedVersion, err := repo.GetVersion(ctx, doc.ID, 1)
	if err != nil {
		t.Fatalf("get saved version: %v", err)
	}
	if savedVersion.DocxStorageKey != docxKey {
		t.Fatalf("docx storage key = %q, want %q", savedVersion.DocxStorageKey, docxKey)
	}

	pdfKey := documentContentStorageKey(doc.ID, 1, "pdf")
	if version.PdfStorageKey != pdfKey {
		t.Fatalf("pdf storage key = %q, want %q", version.PdfStorageKey, pdfKey)
	}
}

func TestExportBrowserContentUsesBrowserDocgenRoute(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	store := documentmemory.NewAttachmentStore()

	doc := seedDraftDocument(t, ctx, repo, now)
	seedCompatiblePOProfileSchemaSet(t, repo)
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:      doc.ID,
		Number:          1,
		Content:         testMDDMBodyUpdated,
		ContentHash:     contentHash(testMDDMBodyUpdated),
		ChangeSummary:   "Initial version",
		ContentSource:   domain.ContentSourceBrowserEditor,
		TextContent:     plainTextFromMDDM(testMDDMBodyUpdated),
		TemplateKey:     "po-mddm-canvas",
		TemplateVersion: 1,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	payloadCh := make(chan []byte, 1)
	docgenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate-browser" {
			t.Fatalf("docgen path = %q, want /generate-browser", r.URL.Path)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read docgen payload: %v", err)
		}
		payloadCh <- raw
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		_, _ = w.Write([]byte("docx-binary"))
	}))
	defer docgenServer.Close()

	gotenbergServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forms/libreoffice/convert" {
			t.Fatalf("gotenberg path = %q, want /forms/libreoffice/convert", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer gotenbergServer.Close()

	service := NewService(repo, nil, fixedClock{now: now}).
		WithAttachmentStore(store).
		WithDocgenClient(docgen.NewClient(config.DocgenConfig{
			Enabled:               true,
			APIURL:                docgenServer.URL,
			RequestTimeoutSeconds: 1,
		})).
		WithGotenberg(gotenberg.NewClient(gotenbergServer.URL))

	if _, err := service.RenderContentPDFAuthorized(ctx, doc.ID, "trace-test"); err != nil {
		t.Fatalf("RenderContentPDFAuthorized() error = %v", err)
	}

	select {
	case raw := <-payloadCh:
		// Header is prepended; assert the md-doc-header block and converted MDDM body are both present.
		if !bytes.Contains(raw, []byte(`md-doc-header`)) {
			t.Fatalf("payload missing md-doc-header: %s", raw)
		}
		if !bytes.Contains(raw, []byte(`<p>Atualizado</p>`)) {
			t.Fatalf("payload missing converted MDDM paragraph: %s", raw)
		}
		if bytes.Contains(raw, []byte(`<section><p>Atualizado</p></section>`)) {
			t.Fatalf("payload still contains legacy HTML body: %s", raw)
		}
	default:
		t.Fatal("expected browser docgen payload")
	}
}

func TestSaveNativeContentAuthorizedDeletesDocxWhenPDFConversionFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	store := documentmemory.NewAttachmentStore()

	doc := seedDraftDocument(t, ctx, repo, now)
	seedCompatiblePOProfileSchemaSet(t, repo)

	docgenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate" {
			t.Fatalf("docgen path = %q, want /generate", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("docx-binary"))
	}))
	defer docgenServer.Close()

	service := NewService(repo, nil, fixedClock{now: now}).
		WithAttachmentStore(store).
		WithDocgenClient(docgen.NewClient(config.DocgenConfig{
			Enabled:               true,
			APIURL:                docgenServer.URL,
			RequestTimeoutSeconds: 1,
		}))

	_, err := service.SaveNativeContentAuthorized(ctx, domain.SaveNativeContentCommand{
		DocumentID: doc.ID,
		Content:    map[string]any{},
		TraceID:    "trace-save",
	})
	if err == nil {
		t.Fatal("expected PDF conversion failure")
	}

	docxKey := documentContentStorageKey(doc.ID, 1, "docx")
	if _, err := store.Open(ctx, docxKey); err == nil {
		t.Fatalf("expected cached docx %q to be deleted after conversion failure", docxKey)
	}

	versions, err := repo.ListVersions(ctx, doc.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 0 {
		t.Fatalf("version count = %d, want 0", len(versions))
	}
}

func TestSaveNativeContentAuthorizedIncludesPendingRevisionInDocgenPayload(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	store := documentmemory.NewAttachmentStore()

	doc := seedDraftDocument(t, ctx, repo, now)
	seedCompatiblePOProfileSchemaSet(t, repo)
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "{}",
		ContentHash:   contentHash("{}"),
		ChangeSummary: "Initial version",
		ContentSource: domain.ContentSourceNative,
		NativeContent: map[string]any{},
		TextContent:   "{}",
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	payloadCh := make(chan docgen.RenderPayload, 1)
	docgenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate" {
			t.Fatalf("docgen path = %q, want /generate", r.URL.Path)
		}
		var payload docgen.RenderPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode docgen payload: %v", err)
		}
		payloadCh <- payload
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("docx-binary"))
	}))
	defer docgenServer.Close()

	gotenbergServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/forms/libreoffice/convert" {
			t.Fatalf("gotenberg path = %q, want /forms/libreoffice/convert", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer gotenbergServer.Close()

	service := NewService(repo, nil, fixedClock{now: now}).
		WithAttachmentStore(store).
		WithDocgenClient(docgen.NewClient(config.DocgenConfig{
			Enabled:               true,
			APIURL:                docgenServer.URL,
			RequestTimeoutSeconds: 1,
		})).
		WithGotenberg(gotenberg.NewClient(gotenbergServer.URL))

	version, err := service.SaveNativeContentAuthorized(ctx, domain.SaveNativeContentCommand{
		DocumentID: doc.ID,
		Content:    map[string]any{},
		TraceID:    "trace-save",
	})
	if err != nil {
		t.Fatalf("save native content: %v", err)
	}
	if version.Number != 2 {
		t.Fatalf("saved version number = %d, want 2", version.Number)
	}

	var payload docgen.RenderPayload
	select {
	case payload = <-payloadCh:
	default:
		t.Fatal("expected docgen payload")
	}

	if len(payload.Revisions) != 2 {
		t.Fatalf("revision count = %d, want 2", len(payload.Revisions))
	}
	if payload.Revisions[0].Versao != "1" {
		t.Fatalf("revision[0].versao = %q, want 1", payload.Revisions[0].Versao)
	}
	if payload.Revisions[1].Versao != "2" {
		t.Fatalf("revision[1].versao = %q, want 2", payload.Revisions[1].Versao)
	}
	if payload.Revisions[1].Data != now.Format("2006-01-02") {
		t.Fatalf("revision[1].data = %q, want %q", payload.Revisions[1].Data, now.Format("2006-01-02"))
	}
	if payload.Revisions[1].Descricao != "Content version 2" {
		t.Fatalf("revision[1].descricao = %q, want %q", payload.Revisions[1].Descricao, "Content version 2")
	}
	if payload.Revisions[1].Por != doc.OwnerID {
		t.Fatalf("revision[1].por = %q, want %q", payload.Revisions[1].Por, doc.OwnerID)
	}
}

func TestSaveNativeContentRejectsStaleDraftToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	store := documentmemory.NewAttachmentStore()

	doc := seedDraftDocument(t, ctx, repo, now)
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "{}",
		ContentHash:   contentHash("{}"),
		ChangeSummary: "Initial version",
		ContentSource: domain.ContentSourceNative,
		NativeContent: map[string]any{},
		Values:        map[string]any{},
		TextContent:   "{}",
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save version: %v", err)
	}

	service := NewService(repo, nil, fixedClock{now: now}).WithAttachmentStore(store)

	_, err := service.SaveNativeContentAuthorized(ctx, domain.SaveNativeContentCommand{
		DocumentID: doc.ID,
		DraftToken: "v1:stale",
		Content: map[string]any{
			"identificacaoProcesso": map[string]any{"objetivo": "novo objetivo"},
		},
		TraceID: "trace-stale",
	})
	if !errors.Is(err, domain.ErrDraftConflict) {
		t.Fatalf("err = %v, want ErrDraftConflict", err)
	}
}

func TestResolveDocumentTemplatePrefersDocumentAssignmentOverProfileDefault(t *testing.T) {
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, nil)
	seedCompatiblePOProfileSchemaSet(t, repo)

	if err := repo.UpsertDocumentTemplateAssignment(context.Background(), domain.DocumentTemplateAssignment{
		DocumentID:      "doc-1",
		TemplateKey:     "po-doc-special",
		TemplateVersion: 2,
		AssignedAt:      time.Unix(0, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert assignment: %v", err)
	}
	if err := repo.UpsertDocumentTemplateVersionForTest(context.Background(), domain.DocumentTemplateVersion{
		TemplateKey:   "po-doc-special",
		Version:       2,
		ProfileCode:   "po",
		SchemaVersion: 3,
		Name:          "PO special canvas",
		Definition:    map[string]any{"type": "page", "id": "special"},
		CreatedAt:     time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatalf("upsert template version: %v", err)
	}

	got, err := service.ResolveDocumentTemplate(context.Background(), "doc-1", "po")
	if err != nil {
		t.Fatalf("resolve template: %v", err)
	}
	if got.TemplateKey != "po-doc-special" || got.Version != 2 {
		t.Fatalf("resolved template = %+v, want po-doc-special v2", got)
	}
}

func seedCompatiblePOProfileSchemaSet(t *testing.T, repo *documentmemory.Repository) {
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

	if err := repo.UpsertDocumentTypeDefinition(context.Background(), domain.DocumentTypeDefinition{
		Key:           "po",
		Name:          "Procedimento Operacional",
		ActiveVersion: 1,
		Schema: domain.DocumentTypeSchema{
			Sections: []domain.SectionDef{
				{
					Key:   "identificacaoProcesso",
					Num:   "2",
					Title: "Identificacao do Processo",
					Fields: []domain.FieldDef{
						{Key: "objetivo", Label: "Objetivo", Type: "textarea"},
					},
				},
				{
					Key:   "visaoGeral",
					Num:   "4",
					Title: "Visao Geral do Processo",
					Fields: []domain.FieldDef{
						{Key: "descricaoProcesso", Label: "Descricao do processo", Type: "rich"},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("seed document type definition: %v", err)
	}

	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
		ProfileCode:   "po",
		Version:       1,
		IsActive:      true,
		ContentSchema: schema,
	}); err != nil {
		t.Fatalf("seed profile schema v1: %v", err)
	}
	if err := repo.UpsertDocumentProfileSchemaVersion(context.Background(), domain.DocumentProfileSchemaVersion{
		ProfileCode:   "po",
		Version:       3,
		IsActive:      false,
		ContentSchema: schema,
	}); err != nil {
		t.Fatalf("seed profile schema v3: %v", err)
	}
}

func seedDraftDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, now time.Time) domain.Document {
	t.Helper()

	doc := domain.Document{
		ID:                   "doc-1",
		Title:                "MetalDocs Procedure",
		DocumentType:         "po",
		DocumentProfile:      "po",
		DocumentFamily:       "procedure",
		DocumentSequence:     1,
		DocumentCode:         "PO-001",
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
	return doc
}
