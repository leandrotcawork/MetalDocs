package httpdelivery

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

func (h *Handler) handleDocumentRuntimeContentPut(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	userID := strings.TrimSpace(authn.UserIDFromContext(r.Context()))
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	values := payload
	if rawValues, ok := payload["values"]; ok {
		if typed, ok := rawValues.(map[string]any); ok {
			values = typed
		} else {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid values payload", traceID)
			return
		}
	}

	_, err := h.service.SaveDocumentValuesAuthorized(r.Context(), domain.SaveDocumentValuesCommand{
		DocumentID: documentID,
		Values:     values,
		TraceID:    traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
