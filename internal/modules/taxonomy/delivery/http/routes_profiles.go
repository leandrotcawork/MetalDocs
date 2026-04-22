package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/taxonomy/domain"
	"metaldocs/internal/platform/authn"
)

const defaultTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

type profileUpsertRequest struct {
	Code                     string  `json:"code"`
	FamilyCode               string  `json:"familyCode"`
	Name                     string  `json:"name"`
	Description              string  `json:"description"`
	ReviewIntervalDays       int     `json:"reviewIntervalDays"`
	DefaultTemplateVersionID *string `json:"defaultTemplateVersionId"`
	OwnerUserID              *string `json:"ownerUserId"`
	EditableByRole           string  `json:"editableByRole"`
}

type setDefaultTemplateRequest struct {
	TemplateVersionID string `json:"templateVersionId"`
}

func (h *Handler) listProfiles(w http.ResponseWriter, r *http.Request) {
	includeArchived, err := parseIncludeArchived(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "includeArchived must be true or false")
		return
	}

	items, err := h.profiles.List(r.Context(), tenantIDFromRequest(r), includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list profiles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createProfile(w http.ResponseWriter, r *http.Request) {
	var req profileUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON payload")
		return
	}

	profile := &domain.DocumentProfile{
		Code:                     strings.TrimSpace(req.Code),
		TenantID:                 tenantIDFromRequest(r),
		FamilyCode:               strings.TrimSpace(req.FamilyCode),
		Name:                     strings.TrimSpace(req.Name),
		Description:              strings.TrimSpace(req.Description),
		ReviewIntervalDays:       req.ReviewIntervalDays,
		DefaultTemplateVersionID: req.DefaultTemplateVersionID,
		OwnerUserID:              req.OwnerUserID,
		EditableByRole:           strings.TrimSpace(req.EditableByRole),
	}
	if profile.Code == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "code is required")
		return
	}

	if err := h.profiles.Create(r.Context(), profile); err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.profiles.Get(r.Context(), tenantIDFromRequest(r), r.PathValue("code"))
	if err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON payload")
		return
	}

	profile := &domain.DocumentProfile{
		Code:                     r.PathValue("code"),
		TenantID:                 tenantIDFromRequest(r),
		FamilyCode:               strings.TrimSpace(req.FamilyCode),
		Name:                     strings.TrimSpace(req.Name),
		Description:              strings.TrimSpace(req.Description),
		ReviewIntervalDays:       req.ReviewIntervalDays,
		DefaultTemplateVersionID: req.DefaultTemplateVersionID,
		OwnerUserID:              req.OwnerUserID,
		EditableByRole:           strings.TrimSpace(req.EditableByRole),
	}
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) setDefaultTemplate(w http.ResponseWriter, r *http.Request) {
	var req setDefaultTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON payload")
		return
	}
	if strings.TrimSpace(req.TemplateVersionID) == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "templateVersionId is required")
		return
	}

	if err := h.profiles.SetDefaultTemplate(
		r.Context(),
		tenantIDFromRequest(r),
		r.PathValue("code"),
		req.TemplateVersionID,
		authn.UserIDFromContext(r.Context()),
	); err != nil {
		h.writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) archiveProfile(w http.ResponseWriter, r *http.Request) {
	if err := h.profiles.Archive(
		r.Context(),
		tenantIDFromRequest(r),
		r.PathValue("code"),
		authn.UserIDFromContext(r.Context()),
	); err != nil {
		h.writeProfileError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "profile not found")
	case errors.Is(err, domain.ErrProfileArchived):
		writeError(w, http.StatusConflict, "PROFILE_ARCHIVED", "profile is archived")
	case errors.Is(err, domain.ErrTemplateNotPublished):
		writeError(w, http.StatusConflict, "TEMPLATE_NOT_PUBLISHED", "template version is not published")
	case errors.Is(err, domain.ErrTemplateProfileMismatch):
		writeError(w, http.StatusConflict, "TEMPLATE_PROFILE_MISMATCH", "template version belongs to different profile")
	case errors.Is(err, domain.ErrProfileCodeImmutable):
		writeError(w, http.StatusBadRequest, "PROFILE_CODE_IMMUTABLE", "profile code is immutable")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func tenantIDFromRequest(r *http.Request) string {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		return defaultTenantID
	}
	return tenantID
}

func parseIncludeArchived(r *http.Request) (bool, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("includeArchived"))
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
