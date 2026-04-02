package application

import (
	"context"
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

func TestSaveNativeContentAuthorizedDeletesDocxWhenPDFConversionFails(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	store := documentmemory.NewAttachmentStore()

	doc := seedDraftDocument(t, ctx, repo, now)

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
