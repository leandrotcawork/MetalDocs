package httpdelivery

import (
	"encoding/json"
	"net/http"
)

type ck5TemplateDraftRequest struct {
	ContentHTML string         `json:"contentHtml"`
	Manifest    map[string]any `json:"manifest"`
}

type ck5TemplateDraftResponse struct {
	ContentHTML string         `json:"contentHtml"`
	Manifest    map[string]any `json:"manifest"`
}

// handleGetCK5TemplateDraft serves GET /api/v1/templates/{key}/ck5-draft.
func (h *Handler) handleGetCK5TemplateDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	if userIDFromContext(r.Context()) == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	html, manifest, err := h.service.GetCK5TemplateDraftContent(r.Context(), key)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, ck5TemplateDraftResponse{
		ContentHTML: html,
		Manifest:    manifest,
	})
}

// handlePutCK5TemplateDraft serves PUT /api/v1/templates/{key}/ck5-draft.
func (h *Handler) handlePutCK5TemplateDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	if userIDFromContext(r.Context()) == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentContentPayloadBytes)

	var req ck5TemplateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	if err := h.service.SaveCK5TemplateDraftAuthorized(r.Context(), key, req.ContentHTML, req.Manifest); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.WriteHeader(http.StatusOK)
}
