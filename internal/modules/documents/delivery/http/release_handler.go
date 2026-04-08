package httpdelivery

import (
	"context"
	"net/http"
	"strings"

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
	userID := userIDFromContext(r.Context())
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

func newTestReleaseHandler(t interface{ Helper() }) *ReleaseHandler {
	t.Helper()
	return NewReleaseHandler(nil)
}

func userIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return strings.TrimSpace(authn.UserIDFromContext(ctx))
}

func extractDocIDFromPath(path string) string {
	trimmed := strings.TrimSpace(path)
	trimmed = strings.TrimPrefix(trimmed, "/api/v1/documents/")
	trimmed = strings.TrimPrefix(trimmed, "/api/documents/")
	trimmed = strings.TrimSuffix(trimmed, "/release")
	trimmed = strings.Trim(trimmed, "/")
	return strings.TrimSpace(trimmed)
}
