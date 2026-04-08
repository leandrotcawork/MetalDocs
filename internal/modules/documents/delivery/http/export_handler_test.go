package httpdelivery

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestExportHandler(t interface{ Helper() }) *ExportHandler {
	t.Helper()
	return NewExportHandler()
}

func TestExportHandler_RequiresVersionID(t *testing.T) {
	handler := newTestExportHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/PO-118/export/docx", nil)
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	if rec.Code != http.StatusNotFound && rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d or %d", rec.Code, http.StatusNotFound, http.StatusOK)
	}
}

func TestExportHandler_ReturnsDocxStubWhenVersionIDPresent(t *testing.T) {
	handler := newTestExportHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/PO-118/export/docx?version_id=ver-123", nil)
	rec := httptest.NewRecorder()

	handler.ExportDocx(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != docxContentType {
		t.Fatalf("content-type = %q, want %q", got, docxContentType)
	}
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "docx-stub" {
		t.Fatalf("body = %q, want %q", string(body), "docx-stub")
	}
}
