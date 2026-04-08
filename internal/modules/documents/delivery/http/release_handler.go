package httpdelivery

import (
	"net/http"

	"metaldocs/internal/platform/authn"
)

type ReleaseHandler struct {
	authChecker ReleaseAuthChecker
}

type ReleaseAuthChecker interface {
	CanApprove(userID, documentID string) bool
}

func NewReleaseHandler(auth ReleaseAuthChecker) *ReleaseHandler {
	return &ReleaseHandler{authChecker: auth}
}

func (h *ReleaseHandler) Release(w http.ResponseWriter, r *http.Request) {
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", requestTraceID(r))
		return
	}

	documentID := extractDocIDFromPath(r.URL.Path)
	if h.authChecker == nil || !h.authChecker.CanApprove(userID, documentID) {
		writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Approval permission required", requestTraceID(r))
		return
	}

	// TODO: real wiring (later task) calls ReleaseService.ReleaseDraft.
	w.WriteHeader(http.StatusOK)
}
