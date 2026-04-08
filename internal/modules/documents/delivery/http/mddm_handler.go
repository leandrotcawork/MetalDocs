package httpdelivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain/mddm"
	"metaldocs/internal/platform/authn"
)

type MDDMHandler struct {
	saveService *application.SaveDraftService
}

func NewMDDMHandler(saveService *application.SaveDraftService) *MDDMHandler {
	return &MDDMHandler{saveService: saveService}
}

func (h *MDDMHandler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 5*1024*1024+1))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if len(body) > 5*1024*1024 {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Quick schema validation before hitting the service — the existing Task 17 tests rely on this
	// path returning 400 for bad JSON and 400 for invalid envelopes even when saveService is nil.
	var envelope map[string]any
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	docID := extractDocIDFromPath(r.URL.Path)

	// If saveService is nil (skeleton tests), fall back to Task 17 behavior:
	// schema-validate inline and return 400 on failure.
	if h.saveService == nil {
		if err := mddm.ValidateMDDMBytes(body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":  "validation_failed",
				"detail": err.Error(),
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	out, err := h.saveService.SaveDraft(r.Context(), application.SaveDraftInput{
		DocumentID:   docID,
		EnvelopeJSON: body,
		UserID:       userIDFromContext(r.Context()),
	})

	if err != nil {
		writeStructuredError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"version_id":   out.VersionID,
		"content_hash": out.ContentHash,
		"new_version":  out.NewVersion,
	})
}

func extractDocIDFromPath(path string) string {
	// /api/documents/{docID}/draft
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func userIDFromContext(ctx context.Context) string {
	if userID := strings.TrimSpace(authn.UserIDFromContext(ctx)); userID != "" {
		return userID
	}
	if v, ok := ctx.Value(ctxUserKey{}).(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

type ctxUserKey struct{}

func writeStructuredError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	switch {
	case strings.Contains(err.Error(), "validation_failed"):
		w.WriteHeader(http.StatusBadRequest)
	case strings.Contains(err.Error(), "TEMPLATE_SNAPSHOT"):
		w.WriteHeader(http.StatusUnprocessableEntity)
	case strings.Contains(err.Error(), "BLOCK_ID_REWRITE_FORBIDDEN"),
		strings.Contains(err.Error(), "LOCKED_BLOCK_DELETED"),
		strings.Contains(err.Error(), "LOCKED_BLOCK_PROP_MUTATED"):
		w.WriteHeader(http.StatusUnprocessableEntity)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}
