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
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type mockCK5ExportClient struct {
	renderDocxFn    func(ctx context.Context, html string) ([]byte, error)
	renderPDFHtmlFn func(ctx context.Context, html string) (string, error)
}

func (m *mockCK5ExportClient) RenderDocx(ctx context.Context, html string) ([]byte, error) {
	if m.renderDocxFn == nil {
		return nil, nil
	}
	return m.renderDocxFn(ctx, html)
}

func (m *mockCK5ExportClient) RenderPDFHtml(ctx context.Context, html string) (string, error) {
	if m.renderPDFHtmlFn == nil {
		return "", nil
	}
	return m.renderPDFHtmlFn(ctx, html)
}

func authCK5ExportRequest(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "owner-1", nil))
}

func seedCK5ExportDocument(t *testing.T, ctx context.Context, repo *documentmemory.Repository, withCK5 bool) domain.Document {
	t.Helper()

	now := time.Now().UTC()
	doc := domain.Document{
		ID:                   "doc-ck5-1",
		Title:                "CK5 Export Test Document",
		DocumentType:         "po",
		DocumentProfile:      "po",
		DocumentFamily:       "procedure",
		DocumentSequence:     1,
		DocumentCode:         "PO-1",
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

	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "<p>browser-html</p>",
		ContentHash:   "hash-v1",
		ChangeSummary: "Initial browser draft",
		ContentSource: domain.ContentSourceBrowserEditor,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save browser version: %v", err)
	}

	if withCK5 {
		if err := repo.SaveVersion(ctx, domain.Version{
			DocumentID:    doc.ID,
			Number:        2,
			Content:       "<p>ck5-html</p>",
			ContentHash:   "hash-v2",
			ChangeSummary: "CK5 version",
			ContentSource: domain.ContentSourceCK5Browser,
			CreatedAt:     now.Add(time.Second),
		}); err != nil {
			t.Fatalf("save ck5 version: %v", err)
		}
	}

	return doc
}

// TestCK5ExportDocx_OK_200
func TestCK5ExportDocx_OK_200(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, true)
	client := &mockCK5ExportClient{
		renderDocxFn: func(_ context.Context, html string) ([]byte, error) {
			if html != "<p>ck5-html</p>" {
				t.Fatalf("html = %q, want %q", html, "<p>ck5-html</p>")
			}
			return []byte("docx-bytes"), nil
		},
	}
	handler := NewHandler(svc).WithCK5ExportClient(client)

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/docx")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, ".docx") {
		t.Fatalf("Content-Disposition = %q, want .docx filename", got)
	}
	if got := rec.Body.String(); got != "docx-bytes" {
		t.Fatalf("body = %q, want %q", got, "docx-bytes")
	}
}

// TestCK5ExportDocx_NoCK5Version_404
func TestCK5ExportDocx_NoCK5Version_404(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, false)
	handler := NewHandler(svc).WithCK5ExportClient(&mockCK5ExportClient{})

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/docx")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusNotFound, "DOC_NOT_FOUND")
}

// TestCK5ExportDocx_NoAuth_404
func TestCK5ExportDocx_NoAuth_404(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	handler := NewHandler(svc).WithCK5ExportClient(&mockCK5ExportClient{})

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/non-existent/export/ck5/docx")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusNotFound, "DOC_NOT_FOUND")
}

// TestCK5ExportDocx_Upstream500_502
func TestCK5ExportDocx_Upstream500_502(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, true)
	client := &mockCK5ExportClient{
		renderDocxFn: func(_ context.Context, _ string) ([]byte, error) {
			return nil, &application.CK5ExportError{Status: http.StatusInternalServerError, Body: "upstream failed"}
		},
	}
	handler := NewHandler(svc).WithCK5ExportClient(client)

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/docx")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusBadGateway, "EXPORT_UPSTREAM_ERROR")
}

// TestCK5ExportDocx_Upstream400_422
func TestCK5ExportDocx_Upstream400_422(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, true)
	client := &mockCK5ExportClient{
		renderDocxFn: func(_ context.Context, _ string) ([]byte, error) {
			return nil, &application.CK5ExportError{Status: http.StatusBadRequest, Body: "invalid html"}
		},
	}
	handler := NewHandler(svc).WithCK5ExportClient(client)

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/docx")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusUnprocessableEntity, "EXPORT_ERROR")
}

// TestCK5ExportPdf_OK_200
func TestCK5ExportPdf_OK_200(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, true)
	client := &mockCK5ExportClient{
		renderPDFHtmlFn: func(_ context.Context, html string) (string, error) {
			if html != "<p>ck5-html</p>" {
				t.Fatalf("html = %q, want %q", html, "<p>ck5-html</p>")
			}
			return "<html><body>wrapped</body></html>", nil
		},
	}
	renderer := &fakePdfRenderer{
		result: []byte("pdf-bytes"),
	}
	handler := NewHandler(svc).WithCK5ExportClient(client).WithPDFConverter(renderer)

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/pdf")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/pdf")
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, ".pdf") {
		t.Fatalf("Content-Disposition = %q, want .pdf filename", got)
	}
	if got := rec.Body.String(); got != "pdf-bytes" {
		t.Fatalf("body = %q, want %q", got, "pdf-bytes")
	}
}

// TestCK5ExportPdf_NoAuth_404
func TestCK5ExportPdf_NoAuth_404(t *testing.T) {
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	handler := NewHandler(svc).WithCK5ExportClient(&mockCK5ExportClient{}).WithPDFConverter(&fakePdfRenderer{})

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/non-existent/export/ck5/pdf")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusNotFound, "DOC_NOT_FOUND")
}

// TestCK5ExportPdf_Upstream500_502
func TestCK5ExportPdf_Upstream500_502(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, true)
	client := &mockCK5ExportClient{
		renderPDFHtmlFn: func(_ context.Context, _ string) (string, error) {
			return "", &application.CK5ExportError{Status: http.StatusInternalServerError, Body: "upstream failed"}
		},
	}
	handler := NewHandler(svc).WithCK5ExportClient(client).WithPDFConverter(&fakePdfRenderer{})

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/pdf")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusBadGateway, "EXPORT_UPSTREAM_ERROR")
}

// TestCK5ExportPdf_Upstream400_422
func TestCK5ExportPdf_Upstream400_422(t *testing.T) {
	ctx := context.Background()
	repo := documentmemory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	doc := seedCK5ExportDocument(t, ctx, repo, true)
	client := &mockCK5ExportClient{
		renderPDFHtmlFn: func(_ context.Context, _ string) (string, error) {
			return "", &application.CK5ExportError{Status: http.StatusBadRequest, Body: "invalid html"}
		},
	}
	handler := NewHandler(svc).WithCK5ExportClient(client).WithPDFConverter(&fakePdfRenderer{})

	req := authCK5ExportRequest(http.MethodGet, "/api/v1/documents/"+doc.ID+"/export/ck5/pdf")
	rec := httptest.NewRecorder()
	handler.handleDocumentSubRoutes(rec, req)

	requireAPIError(t, rec, http.StatusUnprocessableEntity, "EXPORT_ERROR")
}
