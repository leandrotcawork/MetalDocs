package httpdelivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/documents/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestLoadHandler_RequiresAuthentication(t *testing.T) {
	handler := NewLoadHandler(application.NewLoadService(&fakeLoadRepo{draft: nil, released: nil}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/PO-118/load", nil)
	rec := httptest.NewRecorder()

	handler.Load(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "AUTH_UNAUTHORIZED" {
		t.Fatalf("error.code = %q, want %q", body.Error.Code, "AUTH_UNAUTHORIZED")
	}
}

func TestLoadHandler_ReturnsDraftWhenPresent(t *testing.T) {
	repo := &fakeLoadRepo{
		draft: &application.LoadVersion{
			DocumentID:  "PO-118",
			Version:     2,
			Status:      "draft",
			Content:     json.RawMessage(`{"x":"draft"}`),
			TemplateKey: "po",
			TemplateVersion: 2,
			ContentHash: "hash-draft",
		},
		released: &application.LoadVersion{
			DocumentID:  "PO-118",
			Version:     1,
			Status:      "released",
			Content:     json.RawMessage(`{"x":"released"}`),
			TemplateKey: "po",
			TemplateVersion: 1,
			ContentHash: "hash-released",
		},
	}
	handler := NewLoadHandler(application.NewLoadService(repo))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/PO-118/load", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Load(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		DocumentID string          `json:"documentId"`
		Version    int             `json:"version"`
		Status     string          `json:"status"`
		Content    json.RawMessage `json:"content"`
		Template struct {
			Key     string `json:"key"`
			Version int    `json:"version"`
		} `json:"template"`
		ContentHash string `json:"contentHash"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.DocumentID != "PO-118" {
		t.Fatalf("documentId = %q, want %q", body.DocumentID, "PO-118")
	}
	if body.Version != 2 {
		t.Fatalf("version = %d, want %d", body.Version, 2)
	}
	if body.Status != "draft" {
		t.Fatalf("status = %q, want %q", body.Status, "draft")
	}
	if string(body.Content) != `{"x":"draft"}` {
		t.Fatalf("content = %s, want %s", string(body.Content), `{"x":"draft"}`)
	}
	if body.Template.Key != "po" {
		t.Fatalf("template.key = %q, want %q", body.Template.Key, "po")
	}
	if body.Template.Version != 2 {
		t.Fatalf("template.version = %d, want %d", body.Template.Version, 2)
	}
	if body.ContentHash != "hash-draft" {
		t.Fatalf("contentHash = %q, want %q", body.ContentHash, "hash-draft")
	}
}

func TestLoadHandler_FallsBackToReleasedWhenDraftMissing(t *testing.T) {
	repo := &fakeLoadRepo{
		draft: nil,
		released: &application.LoadVersion{
			DocumentID:      "PO-118",
			Version:         1,
			Status:          "released",
			Content:         json.RawMessage(`{"x":"released"}`),
			TemplateKey:     "po",
			TemplateVersion: 1,
			ContentHash:     "hash-released",
		},
	}
	handler := NewLoadHandler(application.NewLoadService(repo))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/PO-118/load", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Load(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		DocumentID string          `json:"documentId"`
		Version    int             `json:"version"`
		Status     string          `json:"status"`
		Content    json.RawMessage `json:"content"`
		Template struct {
			Key     string `json:"key"`
			Version int    `json:"version"`
		} `json:"template"`
		ContentHash string `json:"contentHash"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.DocumentID != "PO-118" {
		t.Fatalf("documentId = %q, want %q", body.DocumentID, "PO-118")
	}
	if body.Version != 1 {
		t.Fatalf("version = %d, want %d", body.Version, 1)
	}
	if body.Status != "released" {
		t.Fatalf("status = %q, want %q", body.Status, "released")
	}
	if string(body.Content) != `{"x":"released"}` {
		t.Fatalf("content = %s, want %s", string(body.Content), `{"x":"released"}`)
	}
	if body.Template.Key != "po" {
		t.Fatalf("template.key = %q, want %q", body.Template.Key, "po")
	}
	if body.Template.Version != 1 {
		t.Fatalf("template.version = %d, want %d", body.Template.Version, 1)
	}
	if body.ContentHash != "hash-released" {
		t.Fatalf("contentHash = %q, want %q", body.ContentHash, "hash-released")
	}
}

func TestLoadHandler_ReturnsNotFoundWhenMissing(t *testing.T) {
	handler := NewLoadHandler(application.NewLoadService(&fakeLoadRepo{draft: nil, released: nil}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/PO-118/load", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Load(rec, req)

	requireAPIError(t, rec, http.StatusNotFound, "DOC_NOT_FOUND")
}

func TestLoadHandler_MapsUnexpectedErrorToInternal(t *testing.T) {
	handler := NewLoadHandler(application.NewLoadService(&fakeLoadRepo{draftErr: errors.New("db unavailable")}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/PO-118/load", nil)
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "user-123", nil))
	rec := httptest.NewRecorder()

	handler.Load(rec, req)

	requireAPIError(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
}

type fakeLoadRepo struct {
	draft    *application.LoadVersion
	released *application.LoadVersion
	draftErr error
	relErr   error
}

func (f *fakeLoadRepo) GetActiveDraft(ctx context.Context, documentID, userID string) (*application.LoadVersion, error) {
	return f.draft, f.draftErr
}

func (f *fakeLoadRepo) GetCurrentReleased(ctx context.Context, documentID string) (*application.LoadVersion, error) {
	return f.released, f.relErr
}
