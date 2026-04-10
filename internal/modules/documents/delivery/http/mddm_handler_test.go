package httpdelivery

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestMDDMHandler(t *testing.T) *MDDMHandler {
	t.Helper()
	return NewMDDMHandler(nil)
}

func TestMDDMHandler_SaveDraft_RejectsInvalidJSON(t *testing.T) {
	handler := newTestMDDMHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/draft", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SaveDraft(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestMDDMHandler_SaveDraft_RejectsInvalidSchema(t *testing.T) {
	handler := newTestMDDMHandler(t)

	body := bytes.NewReader([]byte(`{"mddm_version":1}`))
	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/draft", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SaveDraft(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing required fields, got %d: %s", rec.Code, rec.Body.String())
	}
}
