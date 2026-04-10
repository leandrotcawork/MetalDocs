package httpdelivery

import (
	"bytes"
	"encoding/json"
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

func TestCreateDocumentHandler_CreatesDocumentStub(t *testing.T) {
	handler := newTestCreateHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents", bytes.NewBufferString(`{"template_id":"tpl-1","title":"Documento","profile":"po"}`))
	rec := httptest.NewRecorder()

	handler.CreateDocument(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp createDocumentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "stub" {
		t.Fatalf("id = %q, want %q", resp.ID, "stub")
	}
	if resp.Code != "PO-001" {
		t.Fatalf("code = %q, want %q", resp.Code, "PO-001")
	}
}

func TestCreateDocumentHandler_RejectsMalformedJSON(t *testing.T) {
	handler := newTestCreateHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents", bytes.NewBufferString(`{"template_id":`))
	rec := httptest.NewRecorder()

	handler.CreateDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var envelope apiErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q, want %q", envelope.Error.Code, "VALIDATION_ERROR")
	}
	if envelope.Error.Message != "Invalid JSON payload" {
		t.Fatalf("message = %q, want %q", envelope.Error.Message, "Invalid JSON payload")
	}
}

func TestCreateDocumentHandler_RejectsUnknownFields(t *testing.T) {
	handler := newTestCreateHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/documents", bytes.NewBufferString(`{"template_id":"tpl-1","title":"Documento","profile":"po","unexpected":true}`))
	rec := httptest.NewRecorder()

	handler.CreateDocument(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var envelope apiErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("code = %q, want %q", envelope.Error.Code, "VALIDATION_ERROR")
	}
	if envelope.Error.Message != "Invalid JSON payload" {
		t.Fatalf("message = %q, want %q", envelope.Error.Message, "Invalid JSON payload")
	}
}
