package httpdelivery

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDocumentHandler_ValidatesPayload(t *testing.T) {
	handler := newTestCreateHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents", bytes.NewBufferString("{}"))
	rec := httptest.NewRecorder()

	handler.CreateDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
