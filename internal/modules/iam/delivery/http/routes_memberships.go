package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/platform/authn"
)

const defaultTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

type MembershipHandler struct {
	svc *iamapp.AreaMembershipService
}

type grantMembershipRequest struct {
	UserID   string `json:"userId"`
	AreaCode string `json:"areaCode"`
	Role     string `json:"role"`
}

func NewMembershipHandler(svc *iamapp.AreaMembershipService) *MembershipHandler {
	return &MembershipHandler{svc: svc}
}

func (h *MembershipHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/iam/area-memberships", h.listMemberships)
	mux.HandleFunc("POST /api/v2/iam/area-memberships", h.grantMembership)
	mux.HandleFunc("DELETE /api/v2/iam/area-memberships", h.revokeMembership)
}

func (h *MembershipHandler) listMemberships(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeMembershipAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "Membership service is not configured")
		return
	}

	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if userID == "" {
		userID = strings.TrimSpace(authenticatedActor(r))
	}
	if userID == "" {
		writeMembershipAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "userId is required")
		return
	}

	items, err := h.svc.ListActive(r.Context(), userID, tenantIDFromRequest(r))
	if err != nil {
		writeMembershipAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list memberships")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *MembershipHandler) grantMembership(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeMembershipAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "Membership service is not configured")
		return
	}

	var req grantMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMembershipAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload")
		return
	}
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.AreaCode) == "" || strings.TrimSpace(req.Role) == "" {
		writeMembershipAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "userId, areaCode and role are required")
		return
	}

	grantedBy := authn.UserIDFromContext(r.Context())
	err := h.svc.Grant(
		r.Context(),
		strings.TrimSpace(req.UserID),
		tenantIDFromRequest(r),
		strings.TrimSpace(req.AreaCode),
		iamdomain.Role(strings.ToLower(strings.TrimSpace(req.Role))),
		grantedBy,
	)
	if err != nil {
		h.writeMembershipError(w, err, "Failed to grant membership")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"userId":   strings.TrimSpace(req.UserID),
		"tenantId": tenantIDFromRequest(r),
		"areaCode": strings.TrimSpace(req.AreaCode),
		"role":     strings.ToLower(strings.TrimSpace(req.Role)),
	})
}

func (h *MembershipHandler) revokeMembership(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeMembershipAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "Membership service is not configured")
		return
	}

	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	areaCode := strings.TrimSpace(r.URL.Query().Get("areaCode"))
	if userID == "" || areaCode == "" {
		writeMembershipAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "userId and areaCode are required")
		return
	}
	revokedBy := authn.UserIDFromContext(r.Context())

	err := h.svc.Revoke(r.Context(), userID, tenantIDFromRequest(r), areaCode, revokedBy)
	if err != nil {
		h.writeMembershipError(w, err, "Failed to revoke membership")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MembershipHandler) writeMembershipError(w http.ResponseWriter, err error, defaultMessage string) {
	switch {
	case errors.Is(err, iamapp.ErrMembershipNotFound):
		writeMembershipAPIError(w, http.StatusNotFound, "MEMBERSHIP_NOT_FOUND", "Membership not found")
	case errors.Is(err, iamapp.ErrUnknownRole):
		writeMembershipAPIError(w, http.StatusBadRequest, "UNKNOWN_ROLE", "Unknown role")
	default:
		writeMembershipAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", defaultMessage)
	}
}

func tenantIDFromRequest(r *http.Request) string {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		return defaultTenantID
	}
	return tenantID
}

func writeMembershipAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}
