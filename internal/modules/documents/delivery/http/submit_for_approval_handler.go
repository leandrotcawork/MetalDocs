package httpdelivery

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/platform/authn"
)

type SubmitForApprovalHandler struct {
	service *application.SubmitForApprovalService
}

func NewSubmitForApprovalHandler(service *application.SubmitForApprovalService) *SubmitForApprovalHandler {
	return &SubmitForApprovalHandler{service: service}
}

func (h *SubmitForApprovalHandler) SubmitForApproval(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(authn.UserIDFromContext(r.Context())) == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	rawDraftID := strings.TrimSpace(r.URL.Query().Get("draft_id"))
	draftID, err := uuid.Parse(rawDraftID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid draft_id", traceID)
		return
	}

	if h.service == nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "SubmitForApprovalService is not configured", traceID)
		return
	}

	if err := h.service.SubmitForApproval(r.Context(), draftID); err != nil {
		if errors.Is(err, application.ErrSubmitForApprovalDraftNotDraft) {
			writeAPIError(w, http.StatusUnprocessableEntity, "DOCUMENTS_DRAFT_NOT_DRAFT", "Draft is not in draft status", traceID)
			return
		}
		writeAPIError(w, http.StatusUnprocessableEntity, "SUBMIT_FOR_APPROVAL_FAILED", "Failed to submit draft for approval", traceID)
		return
	}

	w.WriteHeader(http.StatusOK)
}
