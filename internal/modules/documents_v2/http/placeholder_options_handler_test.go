package documentshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	templatesdomain "metaldocs/internal/modules/templates_v2/domain"
)

type fakeOptionsSchemaReader struct {
	phs []templatesdomain.Placeholder
	err error
}

func (f fakeOptionsSchemaReader) LoadPlaceholderSchema(_ context.Context, _, _ string) ([]templatesdomain.Placeholder, error) {
	return f.phs, f.err
}

type fakeOptionsIAMReader struct {
	opts []UserOptionView
	err  error
}

func (f fakeOptionsIAMReader) ListUserOptions(_ context.Context, _ string) ([]UserOptionView, error) {
	return f.opts, f.err
}

func newOptionsReq(docID, pid string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v2/documents/"+docID+"/placeholder-options/"+pid, nil)
	req.SetPathValue("id", docID)
	req.SetPathValue("pid", pid)
	req.Header.Set("X-Tenant-ID", "tenant-1")
	return req
}

func TestPlaceholderOptions_UserType_Returns200WithIAMOptions(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p-user", Type: templatesdomain.PHUser}}
	iamOpts := []UserOptionView{
		{UserID: "u1", DisplayName: "Alice"},
		{UserID: "u2", DisplayName: "Bob"},
	}
	h := NewPlaceholderOptionsHandler(fakeOptionsSchemaReader{phs: schema}, fakeOptionsIAMReader{opts: iamOpts})
	rec := httptest.NewRecorder()
	h.HandleGetOptions(rec, newOptionsReq("doc-1", "p-user"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	opts, _ := body["options"].([]any)
	if len(opts) != 2 {
		t.Errorf("options len=%d, want 2", len(opts))
	}
}

func TestPlaceholderOptions_SelectType_Returns200WithSchemaOptions(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p-sel", Type: templatesdomain.PHSelect, Options: []string{"A", "B", "C"}}}
	h := NewPlaceholderOptionsHandler(fakeOptionsSchemaReader{phs: schema}, fakeOptionsIAMReader{})
	rec := httptest.NewRecorder()
	h.HandleGetOptions(rec, newOptionsReq("doc-1", "p-sel"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	opts, _ := body["options"].([]any)
	if len(opts) != 3 {
		t.Errorf("options len=%d, want 3", len(opts))
	}
}

func TestPlaceholderOptions_TextType_Returns400(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p-text", Type: templatesdomain.PHText}}
	h := NewPlaceholderOptionsHandler(fakeOptionsSchemaReader{phs: schema}, fakeOptionsIAMReader{})
	rec := httptest.NewRecorder()
	h.HandleGetOptions(rec, newOptionsReq("doc-1", "p-text"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	errBody, _ := body["error"].(map[string]any)
	if errBody["code"] != "not_a_choice_placeholder" {
		t.Errorf("code=%v, want not_a_choice_placeholder", errBody["code"])
	}
}

func TestPlaceholderOptions_UnknownPID_Returns400(t *testing.T) {
	schema := []templatesdomain.Placeholder{{ID: "p-text", Type: templatesdomain.PHText}}
	h := NewPlaceholderOptionsHandler(fakeOptionsSchemaReader{phs: schema}, fakeOptionsIAMReader{})
	rec := httptest.NewRecorder()
	h.HandleGetOptions(rec, newOptionsReq("doc-1", "nonexistent"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}
