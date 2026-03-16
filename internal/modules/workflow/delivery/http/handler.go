package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	docdomain "metaldocs/internal/modules/documents/domain"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

type Handler struct {
	service *workflowapp.Service
}

type TransitionRequest struct {
	ToStatus string `json:"toStatus"`
	Reason   string `json:"reason,omitempty"`
}

func NewHandler(service *workflowapp.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/workflow/documents/", h.handleDocumentWorkflowRoutes)
}

func (h *Handler) handleDocumentWorkflowRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	traceID := requestTraceID(r)

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workflow/documents/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "transitions" {
		return
	}

	var req TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	result, err := h.service.Transition(r.Context(), workflowdomain.TransitionCommand{
		DocumentID: parts[0],
		ToStatus:   req.ToStatus,
		ActorID:    actorID,
		Reason:     req.Reason,
		TraceID:    traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"documentId": result.DocumentID,
		"fromStatus": result.FromStatus,
		"toStatus":   result.ToStatus,
	})
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error, traceID string) {
	switch {
	case errors.Is(err, workflowdomain.ErrInvalidCommand):
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data", traceID)
	case errors.Is(err, workflowdomain.ErrInvalidTransition):
		writeAPIError(w, http.StatusConflict, "CONFLICT_ERROR", "Invalid workflow transition", traceID)
	case errors.Is(err, docdomain.ErrDocumentNotFound):
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
	}
}

type apiErrorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
	TraceID string         `json:"trace_id"`
}

func requestTraceID(r *http.Request) string {
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id")); traceID != "" {
		return traceID
	}
	return "trace-local"
}

func writeAPIError(w http.ResponseWriter, status int, code, message, traceID string) {
	writeJSON(w, status, apiErrorEnvelope{
		Error: apiError{
			Code:    code,
			Message: message,
			Details: map[string]any{},
			TraceID: traceID,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
