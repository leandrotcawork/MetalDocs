package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

type ck5ContentRequest struct {
	Body string `json:"body"`
}

type ck5ContentResponse struct {
	Body string `json:"body"`
}

// handleDocumentContentCK5Get serves GET /api/v1/documents/{id}/content/ck5.
func (h *Handler) handleDocumentContentCK5Get(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	if userIDFromContext(r.Context()) == "" {
		h.writeDomainError(w, domain.ErrDocumentNotFound, traceID)
		return
	}
	html, err := h.service.GetCK5DocumentContentAuthorized(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	writeJSON(w, http.StatusOK, ck5ContentResponse{Body: html})
}

// handleDocumentContentCK5Post serves POST /api/v1/documents/{id}/content/ck5.
func (h *Handler) handleDocumentContentCK5Post(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentContentPayloadBytes)

	var req ck5ContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	if strings.TrimSpace(req.Body) == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "body is required", traceID)
		return
	}

	if err := h.service.SaveCK5DocumentContentAuthorized(r.Context(), documentID, req.Body); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
