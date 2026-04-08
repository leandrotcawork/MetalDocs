package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

type LoadHandler struct {
	svc *application.LoadService
}

func NewLoadHandler(svc *application.LoadService) *LoadHandler {
	return &LoadHandler{svc: svc}
}

type loadTemplateResponse struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

type loadDocumentResponse struct {
	DocumentID  string               `json:"documentId"`
	Version     int                  `json:"version"`
	Status      string               `json:"status"`
	Content     json.RawMessage      `json:"content"`
	Template    loadTemplateResponse `json:"template"`
	ContentHash string               `json:"contentHash"`
}

// Load handles GET /api/v1/documents/{id}/load.
func (h *LoadHandler) Load(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	userID := strings.TrimSpace(authn.UserIDFromContext(r.Context()))
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	documentID := extractDocumentIDFromLoadPath(r.URL.Path)
	if documentID == "" {
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
		return
	}

	if h == nil || h.svc == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	out, err := h.svc.LoadForEdit(r.Context(), documentID, userID)
	if errors.Is(err, domain.ErrDocumentNotFound) {
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
		return
	}
	if errors.Is(err, domain.ErrInvalidCommand) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data", traceID)
		return
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}

	writeJSON(w, http.StatusOK, loadDocumentResponse{
		DocumentID:  out.DocumentID,
		Version:     out.Version,
		Status:      out.Status,
		Content:     out.Content,
		Template:    loadTemplateResponse{Key: out.TemplateKey, Version: out.TemplateVersion},
		ContentHash: out.ContentHash,
	})
}

func extractDocumentIDFromLoadPath(path string) string {
	trimmed := strings.TrimSpace(path)
	trimmed = strings.TrimPrefix(trimmed, "/api/v1/documents/")
	if trimmed == path {
		trimmed = strings.TrimPrefix(trimmed, "/api/documents/")
	}
	if !strings.HasSuffix(trimmed, "/load") {
		return ""
	}
	docID := strings.TrimSuffix(trimmed, "/load")
	docID = strings.Trim(docID, "/")
	if docID == "" || strings.Contains(docID, "/") {
		return ""
	}
	return strings.TrimSpace(docID)
}
