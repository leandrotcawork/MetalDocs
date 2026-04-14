package httpdelivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

// ---------------------------------------------------------------------------
// Route dispatchers
// ---------------------------------------------------------------------------

// handleTemplatesCollection handles /api/v1/templates (no trailing slash).
func (h *Handler) handleTemplatesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListTemplates(w, r)
	case http.MethodPost:
		h.handleCreateTemplate(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleTemplatesSubRoutes handles /api/v1/templates/* (trailing slash route).
// Dispatches by path suffix and HTTP method.
func (h *Handler) handleTemplatesSubRoutes(w http.ResponseWriter, r *http.Request) {
	const prefix = "/api/v1/templates/"
	suffix := strings.TrimPrefix(r.URL.Path, prefix)
	// Remove any leading/trailing slashes from suffix for clean splitting.
	suffix = strings.Trim(suffix, "/")
	parts := strings.Split(suffix, "/")

	// parts[0] is the key or "import".
	switch {
	case len(parts) == 1 && parts[0] == "import" && r.Method == http.MethodPost:
		h.handleImportTemplate(w, r)

	case len(parts) == 1 && parts[0] != "":
		// /api/v1/templates/{key}
		key := parts[0]
		switch r.Method {
		case http.MethodGet:
			h.handleGetTemplate(w, r, key)
		case http.MethodDelete:
			h.handleDeleteDraft(w, r, key)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}

	case len(parts) == 2 && parts[0] != "":
		key := parts[0]
		action := parts[1]
		switch action {
		case "draft":
			if r.Method == http.MethodPut {
				h.handleSaveDraft(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "publish":
			if r.Method == http.MethodPost {
				h.handlePublish(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "edit":
			if r.Method == http.MethodPost {
				h.handleEditPublished(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "deprecate":
			if r.Method == http.MethodPost {
				h.handleDeprecate(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "clone":
			if r.Method == http.MethodPost {
				h.handleClone(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "discard-draft":
			if r.Method == http.MethodPost {
				h.handleDiscardDraft(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "acknowledge-stripped":
			if r.Method == http.MethodPost {
				h.handleAcknowledgeStripped(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "export":
			if r.Method == http.MethodGet {
				h.handleExportTemplate(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "preview-docx":
			if r.Method == http.MethodPost {
				h.handleTemplatePreviewDocx(w, r, key)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		default:
			http.NotFound(w, r)
		}

	default:
		http.NotFound(w, r)
	}
}

// ---------------------------------------------------------------------------
// Handler implementations
// ---------------------------------------------------------------------------

// handleListTemplates handles GET /api/v1/templates?profileCode=X
func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	profileCode := strings.TrimSpace(r.URL.Query().Get("profileCode"))
	if profileCode == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "profileCode query parameter is required", traceID)
		return
	}

	versions, err := h.service.ListTemplatesByProfile(r.Context(), profileCode)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]templateVersionResponse, 0, len(versions))
	for _, v := range versions {
		out = append(out, templateVersionResponse{
			TemplateKey: v.TemplateKey,
			Version:     v.Version,
			ProfileCode: v.ProfileCode,
			Name:        v.Name,
			Status:      v.Status,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// handleCreateTemplate handles POST /api/v1/templates
func (h *Handler) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req createTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	draft, err := h.service.CreateDraftAuthorized(r.Context(), req.ProfileCode, req.Name, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, templateDraftToResponse(draft))
}

// handleGetTemplate handles GET /api/v1/templates/{key}
// Returns the draft if one exists; otherwise falls back to the latest published version.
func (h *Handler) handleGetTemplate(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	draft, err := h.service.GetTemplateDraft(r.Context(), key)
	if err == nil {
		writeJSON(w, http.StatusOK, templateDraftToResponse(draft))
		return
	}

	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		h.writeDomainError(w, err, traceID)
		return
	}

	// No draft — look for the latest published version.
	latest, latestErr := h.service.GetLatestPublishedTemplate(r.Context(), key)
	if latestErr != nil {
		h.writeDomainError(w, latestErr, traceID)
		return
	}

	writeJSON(w, http.StatusOK, templateVersionResponse{
		TemplateKey: latest.TemplateKey,
		Version:     latest.Version,
		ProfileCode: latest.ProfileCode,
		Name:        latest.Name,
		Status:      latest.Status,
	})
}

// handleSaveDraft handles PUT /api/v1/templates/{key}/draft
func (h *Handler) handleSaveDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req saveDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	draft, err := h.service.SaveDraftAuthorized(r.Context(), key, req.Blocks, req.Theme, req.Meta, req.LockVersion, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, templateDraftToResponse(draft))
}

// handlePublish handles POST /api/v1/templates/{key}/publish
func (h *Handler) handlePublish(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	tv, err := h.service.PublishAuthorized(r.Context(), key, req.LockVersion, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, templateVersionResponse{
		TemplateKey: tv.TemplateKey,
		Version:     tv.Version,
		ProfileCode: tv.ProfileCode,
		Name:        tv.Name,
		Status:      tv.Status,
	})
}

// handleEditPublished handles POST /api/v1/templates/{key}/edit
func (h *Handler) handleEditPublished(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	draft, err := h.service.EditPublishedAuthorized(r.Context(), key, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, templateDraftToResponse(draft))
}

// handleDeprecate handles POST /api/v1/templates/{key}/deprecate
func (h *Handler) handleDeprecate(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req deprecateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	if req.Version <= 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "version is required", traceID)
		return
	}

	if err := h.service.DeprecateAuthorized(r.Context(), key, req.Version, userID); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleClone handles POST /api/v1/templates/{key}/clone
func (h *Handler) handleClone(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req cloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	draft, err := h.service.CloneAuthorized(r.Context(), key, req.NewName, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, templateDraftToResponse(draft))
}

// handleDiscardDraft handles POST /api/v1/templates/{key}/discard-draft
func (h *Handler) handleDiscardDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	if err := h.service.DiscardDraftAuthorized(r.Context(), key, userID); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteDraft handles DELETE /api/v1/templates/{key}
func (h *Handler) handleDeleteDraft(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	if err := h.service.DeleteDraftAuthorized(r.Context(), key, userID); err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleAcknowledgeStripped handles POST /api/v1/templates/{key}/acknowledge-stripped
func (h *Handler) handleAcknowledgeStripped(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req acknowledgeStrippedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	draft, err := h.service.AcknowledgeStrippedFieldsAuthorized(r.Context(), key, req.LockVersion, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusOK, templateDraftToResponse(draft))
}

// handleExportTemplate handles GET /api/v1/templates/{key}/export?version=N
func (h *Handler) handleExportTemplate(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	versionStr := strings.TrimSpace(r.URL.Query().Get("version"))
	if versionStr == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "version query parameter is required", traceID)
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil || version <= 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "version must be a positive integer", traceID)
		return
	}

	data, err := h.service.ExportTemplate(r.Context(), key, version, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="template-`+key+`-v`+versionStr+`.json"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// handleTemplatePreviewDocx handles POST /api/v1/templates/{key}/preview-docx.
// Renders a .docx from the current draft's blocks and returns the file for download.
func (h *Handler) handleTemplatePreviewDocx(w http.ResponseWriter, r *http.Request, key string) {
	traceID := requestTraceID(r)
	userID := strings.TrimSpace(authn.UserIDFromContext(r.Context()))
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	docxBytes, err := h.service.PreviewDocxAuthorized(r.Context(), key, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-preview.docx"`, key))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docxBytes)
}

// handleImportTemplate handles POST /api/v1/templates/import
func (h *Handler) handleImportTemplate(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	userID := authn.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	profileCode := strings.TrimSpace(r.URL.Query().Get("profileCode"))
	if profileCode == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "profileCode query parameter is required", traceID)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024+1))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to read request body", traceID)
		return
	}
	if len(body) > 10*1024*1024 {
		writeAPIError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Import payload too large", traceID)
		return
	}

	draft, err := h.service.ImportTemplateAuthorized(r.Context(), profileCode, body, userID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, templateDraftToResponse(draft))
}

// ---------------------------------------------------------------------------
// Request / Response DTOs
// ---------------------------------------------------------------------------

type createTemplateRequest struct {
	ProfileCode string `json:"profileCode"`
	Name        string `json:"name"`
}

type saveDraftRequest struct {
	Blocks      json.RawMessage `json:"blocks"`
	Theme       json.RawMessage `json:"theme"`
	Meta        json.RawMessage `json:"meta"`
	LockVersion int             `json:"lockVersion"`
}

type publishRequest struct {
	LockVersion int `json:"lockVersion"`
}

type deprecateRequest struct {
	Version int `json:"version"`
}

type cloneRequest struct {
	NewName string `json:"newName"`
}

type acknowledgeStrippedRequest struct {
	LockVersion int `json:"lockVersion"`
}

type templateDraftResponse struct {
	TemplateKey       string          `json:"templateKey"`
	ProfileCode       string          `json:"profileCode"`
	Name              string          `json:"name"`
	Status            string          `json:"status"`
	LockVersion       int             `json:"lockVersion"`
	HasStrippedFields bool            `json:"hasStrippedFields"`
	Blocks            json.RawMessage `json:"blocks"`
	Theme             json.RawMessage `json:"theme,omitempty"`
	Meta              json.RawMessage `json:"meta,omitempty"`
	UpdatedAt         string          `json:"updatedAt"`
}

type templateVersionResponse struct {
	TemplateKey string `json:"templateKey"`
	Version     int    `json:"version"`
	ProfileCode string `json:"profileCode"`
	Name        string `json:"name"`
	Status      string `json:"status"`
}

type publishValidationErrorResponse struct {
	Errors []publishErrorItem `json:"errors"`
	Error  apiError           `json:"error"`
}

type publishErrorItem struct {
	BlockID   string `json:"blockId"`
	BlockType string `json:"blockType"`
	Field     string `json:"field"`
	Reason    string `json:"reason"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func templateDraftToResponse(d *domain.TemplateDraft) templateDraftResponse {
	return templateDraftResponse{
		TemplateKey:       d.TemplateKey,
		ProfileCode:       d.ProfileCode,
		Name:              d.Name,
		Status:            string(domain.TemplateStatusDraft),
		LockVersion:       d.LockVersion,
		HasStrippedFields: d.HasStrippedFields,
		Blocks:            d.BlocksJSON,
		Theme:             d.ThemeJSON,
		Meta:              d.MetaJSON,
		UpdatedAt:         d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
