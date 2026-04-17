package httpdelivery

import (
	"errors"
	"net/http"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

func (h *Handler) handleTemplateCK5SubmitReview(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}
	if !hasTemplateCapability(r, domain.CapabilityTemplateEdit) {
		writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Insufficient permissions", traceID)
		return
	}

	err := h.service.PublishTemplateForReview(r.Context(), key)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": string(domain.TemplateStatusPendingReview)})
	case errors.Is(err, domain.ErrTemplateNotFound):
		writeAPIError(w, http.StatusNotFound, "TEMPLATE_NOT_FOUND", err.Error(), traceID)
	case errors.Is(err, domain.ErrInvalidTemplateDraftStatus):
		writeAPIError(w, http.StatusConflict, "TEMPLATE_INVALID_DRAFT_STATUS", err.Error(), traceID)
	case errors.Is(err, domain.ErrEmptyTemplateContent):
		writeAPIError(w, http.StatusBadRequest, "TEMPLATE_EMPTY_CONTENT", err.Error(), traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
	}
}

func (h *Handler) handleTemplateCK5Approve(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}
	if !hasTemplateCapability(r, domain.CapabilityTemplatePublish) {
		writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Insufficient permissions", traceID)
		return
	}

	err := h.service.ApproveTemplate(r.Context(), key)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": string(domain.TemplateStatusPublished)})
	case errors.Is(err, domain.ErrTemplateNotFound):
		writeAPIError(w, http.StatusNotFound, "TEMPLATE_NOT_FOUND", err.Error(), traceID)
	case errors.Is(err, domain.ErrInvalidTemplateDraftStatus):
		writeAPIError(w, http.StatusConflict, "TEMPLATE_INVALID_DRAFT_STATUS", err.Error(), traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
	}
}

func hasTemplateCapability(r *http.Request, capability string) bool {
	roles := authn.RolesFromContext(r.Context())
	if len(roles) == 0 {
		return false
	}

	roleCapabilities := map[string]map[string]bool{
		"admin": {
			domain.CapabilityTemplateView:    true,
			domain.CapabilityTemplateEdit:    true,
			domain.CapabilityTemplatePublish: true,
			domain.CapabilityTemplateExport:  true,
		},
		"editor": {
			domain.CapabilityTemplateView:   true,
			domain.CapabilityTemplateExport: true,
		},
		"reviewer": {
			domain.CapabilityTemplateView:   true,
			domain.CapabilityTemplateExport: true,
		},
		"viewer": {},
	}

	for _, role := range roles {
		if caps, ok := roleCapabilities[role]; ok && caps[capability] {
			return true
		}
	}
	return false
}
