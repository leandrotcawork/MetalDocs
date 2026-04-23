package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"metaldocs/internal/modules/taxonomy/domain"
	"metaldocs/internal/platform/authn"
)

type areaUpsertRequest struct {
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Description         string  `json:"description"`
	ParentCode          *string `json:"parentCode"`
	OwnerUserID         *string `json:"ownerUserId"`
	DefaultApproverRole *string `json:"defaultApproverRole"`
}

func (h *Handler) listAreas(w http.ResponseWriter, r *http.Request) {
	includeArchived, err := parseIncludeArchived(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "includeArchived must be true or false")
		return
	}

	items, err := h.areas.List(r.Context(), tenantIDFromRequest(r), includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list areas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createArea(w http.ResponseWriter, r *http.Request) {
	var req areaUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON payload")
		return
	}

	area := &domain.ProcessArea{
		Code:                strings.TrimSpace(req.Code),
		TenantID:            tenantIDFromRequest(r),
		Name:                strings.TrimSpace(req.Name),
		Description:         strings.TrimSpace(req.Description),
		ParentCode:          req.ParentCode,
		OwnerUserID:         req.OwnerUserID,
		DefaultApproverRole: req.DefaultApproverRole,
	}
	if area.Code == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "code is required")
		return
	}

	if err := h.areas.Create(r.Context(), area); err != nil {
		h.writeAreaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, area)
}

func (h *Handler) getArea(w http.ResponseWriter, r *http.Request) {
	area, err := h.areas.Get(r.Context(), tenantIDFromRequest(r), r.PathValue("code"))
	if err != nil {
		h.writeAreaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, area)
}

func (h *Handler) updateArea(w http.ResponseWriter, r *http.Request) {
	var req areaUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON payload")
		return
	}

	area := &domain.ProcessArea{
		Code:                r.PathValue("code"),
		TenantID:            tenantIDFromRequest(r),
		Name:                strings.TrimSpace(req.Name),
		Description:         strings.TrimSpace(req.Description),
		ParentCode:          req.ParentCode,
		OwnerUserID:         req.OwnerUserID,
		DefaultApproverRole: req.DefaultApproverRole,
	}
	if err := h.areas.Update(r.Context(), area); err != nil {
		h.writeAreaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, area)
}

func (h *Handler) archiveArea(w http.ResponseWriter, r *http.Request) {
	if err := h.areas.Archive(
		r.Context(),
		tenantIDFromRequest(r),
		r.PathValue("code"),
		authn.UserIDFromContext(r.Context()),
	); err != nil {
		h.writeAreaError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeAreaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAreaNotFound):
		writeError(w, http.StatusNotFound, "AREA_NOT_FOUND", "process area not found")
	case errors.Is(err, domain.ErrAreaArchived):
		writeError(w, http.StatusConflict, "AREA_ARCHIVED", "process area is archived")
	case errors.Is(err, domain.ErrAreaParentCycle):
		writeError(w, http.StatusBadRequest, "AREA_PARENT_CYCLE", "area parent assignment creates cycle")
	case errors.Is(err, domain.ErrAreaCodeImmutable):
		writeError(w, http.StatusBadRequest, "AREA_CODE_IMMUTABLE", "area code is immutable")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
