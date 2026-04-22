package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

const defaultTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

type MembershipHandler struct {
	svc *iamapp.AreaMembershipService
}

type grantMembershipRequest struct {
	UserID    string `json:"userId"`
	AreaCode  string `json:"areaCode"`
	Role      string `json:"role"`
	GrantedBy string `json:"grantedBy,omitempty"`
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
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "Membership service is not configured", requestTraceID(r))
		return
	}

	userID := strings.TrimSpace(r.URL.Query().Get("userID"))
	if userID == "" {
		userID = strings.TrimSpace(authenticatedActor(r))
	}
	if userID == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "userID is required", requestTraceID(r))
		return
	}

	items, err := h.svc.ListActive(r.Context(), userID, tenantIDFromRequest(r))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list memberships", requestTraceID(r))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *MembershipHandler) grantMembership(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "Membership service is not configured", requestTraceID(r))
		return
	}

	var req grantMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", requestTraceID(r))
		return
	}
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.AreaCode) == "" || strings.TrimSpace(req.Role) == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "userId, areaCode and role are required", requestTraceID(r))
		return
	}

	grantedBy := strings.TrimSpace(req.GrantedBy)
	if grantedBy == "" {
		grantedBy = authenticatedActor(r)
	}
	err := h.svc.Grant(
		r.Context(),
		strings.TrimSpace(req.UserID),
		tenantIDFromRequest(r),
		strings.TrimSpace(req.AreaCode),
		iamdomain.Role(strings.ToLower(strings.TrimSpace(req.Role))),
		grantedBy,
	)
	if err != nil {
		h.writeMembershipError(w, err, "Failed to grant membership", requestTraceID(r))
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
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "Membership service is not configured", requestTraceID(r))
		return
	}

	userID := strings.TrimSpace(r.URL.Query().Get("userID"))
	areaCode := strings.TrimSpace(r.URL.Query().Get("areaCode"))
	if userID == "" || areaCode == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "userID and areaCode are required", requestTraceID(r))
		return
	}
	revokedBy := strings.TrimSpace(r.URL.Query().Get("revokedBy"))
	if revokedBy == "" {
		revokedBy = authenticatedActor(r)
	}

	err := h.svc.Revoke(r.Context(), userID, tenantIDFromRequest(r), areaCode, revokedBy)
	if err != nil {
		h.writeMembershipError(w, err, "Failed to revoke membership", requestTraceID(r))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MembershipHandler) writeMembershipError(w http.ResponseWriter, err error, defaultMessage, traceID string) {
	switch {
	case errors.Is(err, iamapp.ErrMembershipNotFound):
		writeAPIError(w, http.StatusNotFound, "MEMBERSHIP_NOT_FOUND", "Membership not found", traceID)
	case errors.Is(err, iamapp.ErrUnknownRole):
		writeAPIError(w, http.StatusBadRequest, "UNKNOWN_ROLE", "Unknown role", traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", defaultMessage, traceID)
	}
}

func tenantIDFromRequest(r *http.Request) string {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		return defaultTenantID
	}
	return tenantID
}
