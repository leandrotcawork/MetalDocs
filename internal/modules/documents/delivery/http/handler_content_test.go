package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDocumentProfileTemplateDocxDeprecated(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/document-profiles/po/template.docx", nil)
	rec := httptest.NewRecorder()

	handler.handleDocumentProfileTemplateDocx(rec, req, "PO")

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}
	want := `{"error":{"code":"TEMPLATE_DEPRECATED","message":"Carbone template rendering has been removed. Use the content builder instead."}}`
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestHandleDocumentTemplateDocxDeprecated(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/doc-123/template.docx", nil)
	rec := httptest.NewRecorder()

	handler.handleDocumentTemplateDocx(rec, req, "doc-123")

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}
	want := `{"error":{"code":"TEMPLATE_DEPRECATED","message":"Carbone template rendering has been removed. Use the content builder instead."}}`
	if got := rec.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
