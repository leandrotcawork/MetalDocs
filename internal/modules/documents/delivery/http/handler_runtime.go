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

type DocumentRuntimeContentRequest struct {
	Values map[string]any `json:"values"`
}

func (h *Handler) handleDocumentRuntimeContentPut(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	userID := strings.TrimSpace(authn.UserIDFromContext(r.Context()))
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req DocumentRuntimeContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	_, err := h.service.SaveDocumentValuesAuthorized(r.Context(), domain.SaveDocumentValuesCommand{
		DocumentID: documentID,
		Values:     req.Values,
		TraceID:    traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleDocumentExportDocx(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)
	userID := strings.TrimSpace(authn.UserIDFromContext(r.Context()))
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	payload, err := h.service.ExportDocumentDocxAuthorized(r.Context(), documentID, traceID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", `attachment; filename="document-`+documentID+`-runtime.docx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
