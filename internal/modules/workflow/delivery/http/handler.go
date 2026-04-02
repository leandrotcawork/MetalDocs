package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	docdomain "metaldocs/internal/modules/documents/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
	workflowapp "metaldocs/internal/modules/workflow/application"
	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

type Handler struct {
	service *workflowapp.Service
}

type TransitionRequest struct {
	ToStatus         string `json:"toStatus"`
	Reason           string `json:"reason,omitempty"`
	AssignedReviewer string `json:"assignedReviewer,omitempty"`
}

type ApprovalResponse struct {
	ApprovalID       string `json:"approvalId"`
	DocumentID       string `json:"documentId"`
	RequestedBy      string `json:"requestedBy"`
	AssignedReviewer string `json:"assignedReviewer"`
	DecisionBy       string `json:"decisionBy,omitempty"`
	Status           string `json:"status"`
	RequestReason    string `json:"requestReason,omitempty"`
	DecisionReason   string `json:"decisionReason,omitempty"`
	RequestedAt      string `json:"requestedAt"`
	DecidedAt        string `json:"decidedAt,omitempty"`
}

func NewHandler(service *workflowapp.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/workflow/documents/", h.handleDocumentWorkflowRoutes)
}

func (h *Handler) handleDocumentWorkflowRoutes(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/workflow/documents/")
	parts := strings.Split(path, "/")
	if strings.TrimSpace(parts[0]) == "" {
		writeAPIError(w, http.StatusNotFound, "WORKFLOW_ROUTE_NOT_FOUND", "Route not found", traceID)
		return
	}

	if len(parts) == 2 && parts[1] == "transitions" {
		if r.Method == http.MethodPost {
			h.handleTransition(w, r, parts[0], traceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(parts) == 2 && parts[1] == "approvals" {
		if r.Method == http.MethodGet {
			h.handleListApprovals(w, r, parts[0], traceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writeAPIError(w, http.StatusNotFound, "WORKFLOW_ROUTE_NOT_FOUND", "Route not found", traceID)
}

func (h *Handler) handleTransition(w http.ResponseWriter, r *http.Request, documentID, traceID string) {
	var req TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	actorID := iamdomain.UserIDFromContext(r.Context())
	result, err := h.service.Transition(r.Context(), workflowdomain.TransitionCommand{
		DocumentID:       documentID,
		ToStatus:         req.ToStatus,
		ActorID:          actorID,
		Reason:           req.Reason,
		AssignedReviewer: req.AssignedReviewer,
		TraceID:          traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"documentId":       result.DocumentID,
		"fromStatus":       result.FromStatus,
		"toStatus":         result.ToStatus,
		"approvalId":       result.ApprovalID,
		"approvalStatus":   result.ApprovalStatus,
		"assignedReviewer": result.AssignedReviewer,
	})
}

func (h *Handler) handleListApprovals(w http.ResponseWriter, r *http.Request, documentID, traceID string) {
	items, err := h.service.ListApprovals(r.Context(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]ApprovalResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ApprovalResponse{
			ApprovalID:       item.ID,
			DocumentID:       item.DocumentID,
			RequestedBy:      item.RequestedBy,
			AssignedReviewer: item.AssignedReviewer,
			DecisionBy:       item.DecisionBy,
			Status:           item.Status,
			RequestReason:    item.RequestReason,
			DecisionReason:   item.DecisionReason,
			RequestedAt:      item.RequestedAt.UTC().Format(time.RFC3339),
			DecidedAt:        formatOptionalTime(item.DecidedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error, traceID string) {
	switch {
	case errors.Is(err, workflowdomain.ErrInvalidCommand):
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data", traceID)
	case errors.Is(err, workflowdomain.ErrInvalidTransition):
		writeAPIError(w, http.StatusConflict, "CONFLICT_ERROR", "Invalid workflow transition", traceID)
	case errors.Is(err, workflowdomain.ErrApprovalNotFound):
		writeAPIError(w, http.StatusNotFound, "WORKFLOW_APPROVAL_NOT_FOUND", "Workflow approval not found", traceID)
	case errors.Is(err, workflowdomain.ErrApprovalReviewerDenied):
		writeAPIError(w, http.StatusForbidden, "WORKFLOW_APPROVAL_FORBIDDEN", "Actor is not assigned reviewer", traceID)
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

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
