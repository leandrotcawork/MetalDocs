package httpdelivery

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

// fakePdfRenderer is a test double for PDFRenderer.
type fakePdfRenderer struct {
	lastHTML []byte
	lastCSS  []byte
	result   []byte
	err      error
}

func (f *fakePdfRenderer) ConvertHTMLToPDF(_ context.Context, html []byte, css []byte) ([]byte, error) {
	f.lastHTML = html
	f.lastCSS = css
	return f.result, f.err
}

// fakeDocAuthz is a test double for DocumentAuthorizer.
type fakeDocAuthz struct {
	notFound map[string]bool
	err      error
}

func (f *fakeDocAuthz) GetDocumentAuthorized(_ context.Context, documentID string) (domain.Document, error) {
	if f.err != nil {
		return domain.Document{}, f.err
	}
	if f.notFound[documentID] {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return domain.Document{ID: documentID}, nil
}

// makeMultipart builds a multipart/form-data body with index.html and optional style.css.
func makeMultipart(t *testing.T, html string, css string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	htmlPart, _ := writer.CreateFormFile("index.html", "index.html")
	_, _ = htmlPart.Write([]byte(html))
	if css != "" {
		cssPart, _ := writer.CreateFormFile("style.css", "style.css")
		_, _ = cssPart.Write([]byte(css))
	}
	_ = writer.Close()
	return &body, writer.FormDataContentType()
}

func authRenderRequest(t *testing.T, method, target string, body *bytes.Buffer, contentType string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, body)
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
}

// TestHandleDocumentRenderPDF_HappyPath verifies PDF bytes returned with correct Content-Type.
func TestHandleDocumentRenderPDF_HappyPath(t *testing.T) {
	renderer := &fakePdfRenderer{result: []byte("%PDF-1.4 hello")}
	authz := &fakeDocAuthz{}
	handler := NewRenderPDFHandler(renderer, authz)

	body, ct := makeMultipart(t, "<html>hi</html>", "body{}")
	req := authRenderRequest(t, http.MethodPost, "/api/v1/documents/doc-1/render/pdf", body, ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "doc-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type = %q, want %q", got, "application/pdf")
	}
	if !bytes.HasPrefix(rec.Body.Bytes(), []byte("%PDF")) {
		t.Fatalf("expected PDF magic bytes in body, got %q", rec.Body.String())
	}
}

// TestHandleDocumentRenderPDF_NoAuth verifies 401 when no user ID in context.
func TestHandleDocumentRenderPDF_NoAuth(t *testing.T) {
	handler := NewRenderPDFHandler(&fakePdfRenderer{}, &fakeDocAuthz{})

	body, ct := makeMultipart(t, "<html>hi</html>", "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/doc-1/render/pdf", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "doc-1")

	requireAPIError(t, rec, http.StatusUnauthorized, "AUTH_UNAUTHORIZED")
}

// TestHandleDocumentRenderPDF_DocumentNotFound verifies 404 when authz denies.
func TestHandleDocumentRenderPDF_DocumentNotFound(t *testing.T) {
	authz := &fakeDocAuthz{notFound: map[string]bool{"doc-missing": true}}
	handler := NewRenderPDFHandler(&fakePdfRenderer{result: []byte("%PDF")}, authz)

	body, ct := makeMultipart(t, "<html>hi</html>", "")
	req := authRenderRequest(t, http.MethodPost, "/api/v1/documents/doc-missing/render/pdf", body, ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "doc-missing")

	requireAPIError(t, rec, http.StatusNotFound, "DOCUMENT_NOT_FOUND")
}

// TestHandleDocumentRenderPDF_PayloadTooLarge verifies 413 when body exceeds MaxPayloadBytes.
func TestHandleDocumentRenderPDF_PayloadTooLarge(t *testing.T) {
	handler := NewRenderPDFHandler(&fakePdfRenderer{result: []byte("%PDF")}, &fakeDocAuthz{})
	handler.MaxPayloadBytes = 100

	// Build a body that exceeds 100 bytes
	bigHTML := strings.Repeat("x", 500)
	body, ct := makeMultipart(t, bigHTML, "")
	req := authRenderRequest(t, http.MethodPost, "/api/v1/documents/doc-1/render/pdf", body, ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "doc-1")

	requireAPIError(t, rec, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}

// TestHandleDocumentRenderPDF_NilRenderer verifies 502 when renderer is nil.
func TestHandleDocumentRenderPDF_NilRenderer(t *testing.T) {
	handler := NewRenderPDFHandler(nil, &fakeDocAuthz{})

	body, ct := makeMultipart(t, "<html>hi</html>", "")
	req := authRenderRequest(t, http.MethodPost, "/api/v1/documents/doc-1/render/pdf", body, ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "doc-1")

	requireAPIError(t, rec, http.StatusBadGateway, "RENDER_UNAVAILABLE")
}

// TestHandleDocumentRenderPDF_RendererError verifies 502 when renderer returns an error.
func TestHandleDocumentRenderPDF_RendererError(t *testing.T) {
	renderer := &fakePdfRenderer{err: errors.New("gotenberg down")}
	handler := NewRenderPDFHandler(renderer, &fakeDocAuthz{})

	body, ct := makeMultipart(t, "<html>hi</html>", "")
	req := authRenderRequest(t, http.MethodPost, "/api/v1/documents/doc-1/render/pdf", body, ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "doc-1")

	requireAPIError(t, rec, http.StatusBadGateway, "RENDER_UPSTREAM_ERROR")
}

// TestHandleDocumentRenderPDF_AuthzInternalErrorReturns500 verifies 500 when authz returns a
// non-domain error (exercises the default: branch in the error switch).
func TestHandleDocumentRenderPDF_AuthzInternalErrorReturns500(t *testing.T) {
	renderer := &fakePdfRenderer{result: []byte("%PDF")}
	authz := &fakeDocAuthz{err: errors.New("db connection failed")}
	handler := NewRenderPDFHandler(renderer, authz)

	body, ct := makeMultipart(t, "<html></html>", "")
	req := authRenderRequest(t, http.MethodPost, "/api/v1/documents/d1/render/pdf", body, ct)
	rec := httptest.NewRecorder()

	handler.HandleRenderPDF(rec, req, "d1")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}
