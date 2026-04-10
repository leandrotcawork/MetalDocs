package httpdelivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/application"
)

type ReleaseServiceCaller interface {
	ReleaseDraft(ctx context.Context, in application.ReleaseInput) error
}

type ReleaseHandler struct {
	authChecker    ReleaseAuthChecker
	releaseService ReleaseServiceCaller
}

type ReleaseAuthChecker interface {
	CanApprove(userID, documentID string) bool
}

func NewReleaseHandler(auth ReleaseAuthChecker) *ReleaseHandler {
	return &ReleaseHandler{authChecker: auth}
}

func (h *ReleaseHandler) WithReleaseService(svc ReleaseServiceCaller) *ReleaseHandler {
	h.releaseService = svc
	return h
}

func (h *ReleaseHandler) Release(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	documentID := extractDocIDFromPath(r.URL.Path)
	if h.authChecker == nil || !h.authChecker.CanApprove(userID, documentID) {
		writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Approval permission required", traceID)
		return
	}

	if h.releaseService == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Release service not available", traceID)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read request body", traceID)
		return
	}

	var req struct {
		DraftID string `json:"draft_id"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON body", traceID)
			return
		}
	}

	draftID, err := uuid.Parse(req.DraftID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid draft_id: must be a UUID", traceID)
		return
	}

	if err := h.releaseService.ReleaseDraft(r.Context(), application.ReleaseInput{
		DocumentID: documentID,
		DraftID:    draftID,
		ApprovedBy: userID,
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "RELEASE_ERROR", err.Error(), traceID)
		return
	}

	w.WriteHeader(http.StatusOK)
}
