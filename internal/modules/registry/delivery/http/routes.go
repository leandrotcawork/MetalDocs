package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/registry/application"
	registrydomain "metaldocs/internal/modules/registry/domain"
	taxonomydomain "metaldocs/internal/modules/taxonomy/domain"
	"metaldocs/internal/platform/authn"
	"metaldocs/internal/platform/httpresponse"
)

const defaultTenantID = "ffffffff-ffff-ffff-ffff-ffffffffffff"

type createDocRequest struct {
	ProfileCode               string  `json:"profileCode"`
	ProcessAreaCode           string  `json:"processAreaCode"`
	DepartmentCode            *string `json:"departmentCode"`
	Title                     string  `json:"title"`
	OwnerUserID               string  `json:"ownerUserId"`
	ManualCode                *string `json:"manualCode"`
	ManualCodeReason          *string `json:"manualCodeReason"`
	OverrideTemplateVersionID *string `json:"overrideTemplateVersionId"`
	OverrideTemplateReason    *string `json:"overrideTemplateReason"`
}

func (h *Handler) listDocs(w http.ResponseWriter, r *http.Request) {
	filter, err := parseFilter(r)
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	items, err := h.svc.List(r.Context(), tenantIDFromRequest(r), filter)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) createDoc(w http.ResponseWriter, r *http.Request) {
	var req createDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON payload")
		return
	}

	doc, err := h.svc.Create(r.Context(), application.CreateControlledDocumentCmd{
		TenantID:                  tenantIDFromRequest(r),
		ProfileCode:               strings.TrimSpace(req.ProfileCode),
		ProcessAreaCode:           strings.TrimSpace(req.ProcessAreaCode),
		DepartmentCode:            req.DepartmentCode,
		Title:                     strings.TrimSpace(req.Title),
		OwnerUserID:               strings.TrimSpace(req.OwnerUserID),
		ActorUserID:               authn.UserIDFromContext(r.Context()),
		ManualCode:                req.ManualCode,
		ManualCodeReason:          req.ManualCodeReason,
		OverrideTemplateVersionID: req.OverrideTemplateVersionID,
		OverrideTemplateReason:    req.OverrideTemplateReason,
	})
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusCreated, doc)
}

func (h *Handler) getDoc(w http.ResponseWriter, r *http.Request) {
	doc, err := h.svc.Get(r.Context(), tenantIDFromRequest(r), r.PathValue("id"))
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, doc)
}

func (h *Handler) obsoleteDoc(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Obsolete(r.Context(), tenantIDFromRequest(r), r.PathValue("id")); err != nil {
		h.writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) supersedeDoc(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Supersede(r.Context(), tenantIDFromRequest(r), r.PathValue("id")); err != nil {
		h.writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, registrydomain.ErrCDNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "CONTROLLED_DOCUMENT_NOT_FOUND", "controlled document not found")
	case errors.Is(err, registrydomain.ErrCDNotActive):
		httpresponse.WriteError(w, http.StatusConflict, "CONTROLLED_DOCUMENT_NOT_ACTIVE", "controlled document is not active")
	case errors.Is(err, registrydomain.ErrCDCodeTaken):
		httpresponse.WriteError(w, http.StatusConflict, "CONTROLLED_DOCUMENT_CODE_TAKEN", "controlled document code already taken")
	case errors.Is(err, registrydomain.ErrCDArchivedCodeReuse):
		httpresponse.WriteError(w, http.StatusConflict, "CONTROLLED_DOCUMENT_CODE_ARCHIVED", "cannot reuse code from archived controlled document")
	case errors.Is(err, registrydomain.ErrManualCodeReasonRequired):
		httpresponse.WriteError(w, http.StatusBadRequest, "MANUAL_CODE_REASON_REQUIRED", "manual code reason is required")
	case errors.Is(err, registrydomain.ErrOverrideReasonRequired):
		httpresponse.WriteError(w, http.StatusBadRequest, "OVERRIDE_REASON_REQUIRED", "override reason is required")
	case errors.Is(err, registrydomain.ErrOverrideTemplateDeleted):
		httpresponse.WriteError(w, http.StatusConflict, "OVERRIDE_TEMPLATE_DELETED", "override template deleted")
	case errors.Is(err, registrydomain.ErrOverrideNotPublished):
		httpresponse.WriteError(w, http.StatusConflict, "OVERRIDE_TEMPLATE_NOT_PUBLISHED", "override template is not published")
	case errors.Is(err, registrydomain.ErrTemplateProfileMismatch):
		httpresponse.WriteError(w, http.StatusConflict, "TEMPLATE_PROFILE_MISMATCH", "template profile mismatch")
	case errors.Is(err, registrydomain.ErrProfileHasNoDefaultTemplate):
		httpresponse.WriteError(w, http.StatusConflict, "PROFILE_NO_DEFAULT_TEMPLATE", "profile has no default template")
	case errors.Is(err, registrydomain.ErrDefaultObsolete):
		httpresponse.WriteError(w, http.StatusConflict, "DEFAULT_TEMPLATE_OBSOLETE", "default template is obsolete")
	case errors.Is(err, taxonomydomain.ErrProfileNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "profile not found")
	case errors.Is(err, taxonomydomain.ErrAreaNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "AREA_NOT_FOUND", "process area not found")
	case errors.Is(err, taxonomydomain.ErrProfileArchived):
		httpresponse.WriteError(w, http.StatusConflict, "PROFILE_ARCHIVED", "profile is archived")
	case errors.Is(err, taxonomydomain.ErrAreaArchived):
		httpresponse.WriteError(w, http.StatusConflict, "AREA_ARCHIVED", "process area is archived")
	default:
		httpresponse.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func tenantIDFromRequest(r *http.Request) string {
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if tenantID == "" {
		return defaultTenantID
	}
	return tenantID
}

func parseFilter(r *http.Request) (application.CDFilter, error) {
	query := r.URL.Query()
	filter := application.CDFilter{}

	if value := strings.TrimSpace(query.Get("profileCode")); value != "" {
		filter.ProfileCode = &value
	}
	if value := strings.TrimSpace(query.Get("processAreaCode")); value != "" {
		filter.ProcessAreaCode = &value
	}
	if value := strings.TrimSpace(query.Get("departmentCode")); value != "" {
		filter.DepartmentCode = &value
	}
	if value := strings.TrimSpace(query.Get("ownerUserId")); value != "" {
		filter.OwnerUserID = &value
	}
	if value := strings.TrimSpace(query.Get("q")); value != "" {
		filter.Query = &value
	}
	if value := strings.TrimSpace(query.Get("status")); value != "" {
		status := registrydomain.CDStatus(value)
		switch status {
		case registrydomain.CDStatusActive, registrydomain.CDStatusObsolete, registrydomain.CDStatusSuperseded:
			filter.Status = &status
		default:
			return application.CDFilter{}, errors.New("invalid status value")
		}
	}
	if value := strings.TrimSpace(query.Get("limit")); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 0 {
			return application.CDFilter{}, errors.New("invalid limit value")
		}
		filter.Limit = limit
	}
	if value := strings.TrimSpace(query.Get("offset")); value != "" {
		offset, err := strconv.Atoi(value)
		if err != nil || offset < 0 {
			return application.CDFilter{}, errors.New("invalid offset value")
		}
		filter.Offset = offset
	}

	return filter, nil
}
