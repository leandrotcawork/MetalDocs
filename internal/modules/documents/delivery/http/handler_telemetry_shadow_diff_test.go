package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/documents/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type fakeShadowDiffRepo struct {
	last *domain.ShadowDiffEvent
	err  error
}

func (f *fakeShadowDiffRepo) Insert(ctx context.Context, event domain.ShadowDiffEvent) error {
	if f.err != nil {
		return f.err
	}
	f.last = &event
	return nil
}

func TestHandleShadowDiff_PersistsEvent(t *testing.T) {
	repo := &fakeShadowDiffRepo{}
	handler := NewShadowDiffHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"document_id":         "doc-1",
		"version_number":      3,
		"user_id_hash":        "uhash",
		"current_xml_hash":    "chash",
		"shadow_xml_hash":     "shash",
		"diff_summary":        map[string]any{"blocks_equal": 10},
		"current_duration_ms": 500,
		"shadow_duration_ms":  800,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "u-1", nil))

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if repo.last == nil {
		t.Fatalf("expected repo to receive an insert")
	}
	if repo.last.DocumentID != "doc-1" || repo.last.VersionNumber != 3 {
		t.Fatalf("event fields mismatch: %+v", repo.last)
	}
}

func TestHandleShadowDiff_Unauthenticated(t *testing.T) {
	repo := &fakeShadowDiffRepo{}
	handler := NewShadowDiffHandler(repo)

	body, _ := json.Marshal(map[string]any{"document_id": "d", "version_number": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleShadowDiff_RejectsMalformedBody(t *testing.T) {
	repo := &fakeShadowDiffRepo{}
	handler := NewShadowDiffHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "u-1", nil))

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleShadowDiff_RejectsNegativeDurations(t *testing.T) {
	repo := &fakeShadowDiffRepo{}
	handler := NewShadowDiffHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"document_id":         "doc-1",
		"version_number":      1,
		"current_xml_hash":    "h1",
		"shadow_xml_hash":     "h2",
		"current_duration_ms": -1,
		"shadow_duration_ms":  500,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "u-1", nil))

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative duration, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleShadowDiff_RejectsSuccessWithEmptyHashes(t *testing.T) {
	repo := &fakeShadowDiffRepo{}
	handler := NewShadowDiffHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"document_id":         "doc-1",
		"version_number":      1,
		"current_xml_hash":    "", // empty hash, no shadow_error = invalid
		"shadow_xml_hash":     "",
		"current_duration_ms": 500,
		"shadow_duration_ms":  800,
		// shadow_error omitted (empty)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "u-1", nil))

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty hashes on success row, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleShadowDiff_AllowsFailureRowWithEmptyHashes(t *testing.T) {
	repo := &fakeShadowDiffRepo{}
	handler := NewShadowDiffHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"document_id":         "doc-1",
		"version_number":      1,
		"current_xml_hash":    "",
		"shadow_xml_hash":     "",
		"current_duration_ms": 500,
		"shadow_duration_ms":  0,
		"shadow_error":        "worker timeout", // failure row — empty hashes OK
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/mddm-shadow-diff", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "u-1", nil))

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for failure row with empty hashes, got %d: %s", w.Code, w.Body.String())
	}
}
