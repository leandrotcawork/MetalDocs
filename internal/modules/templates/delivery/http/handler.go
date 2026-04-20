package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"metaldocs/internal/modules/templates/application"
	"metaldocs/internal/modules/templates/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

const (
	roleAdmin          = "admin"
	roleTemplateAuthor = "template_author"
	rolePublisher      = "template_publisher"
)

func requireRole(r *http.Request, want ...string) bool {
	roles := iamdomain.RolesFromContext(r.Context())
	for _, role := range roles {
		for _, w := range want {
			if string(role) == w {
				return true
			}
		}
	}
	// fallback: legacy X-User-Roles header (spike/dev bypass)
	hdr := r.Header.Get("X-User-Roles")
	if hdr == "" {
		return false
	}
	for _, w := range want {
		for _, g := range strings.Split(hdr, ",") {
			if strings.TrimSpace(g) == w {
				return true
			}
		}
	}
	return false
}

type Service interface {
	CreateTemplate(ctx context.Context, cmd application.CreateTemplateCmd) (*domain.Template, *domain.TemplateVersion, error)
	SaveDraft(ctx context.Context, cmd application.SaveDraftCmd) error
	PublishVersion(ctx context.Context, cmd application.PublishCmd) (application.PublishResult, error)
	ListTemplates(ctx context.Context, tenantID string) ([]domain.TemplateListItem, error)
	GetVersion(ctx context.Context, templateID string, versionNum int) (*domain.Template, *domain.TemplateVersion, error)
	PresignDocxUpload(ctx context.Context, templateID string, versionNum int) (url, storageKey string, err error)
	PresignSchemaUpload(ctx context.Context, templateID string, versionNum int) (url, storageKey string, err error)
	PresignObjectDownload(ctx context.Context, storageKey string) (url string, err error)
}

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/templates", h.listTemplates)
	mux.HandleFunc("POST /api/v2/templates", h.createTemplate)
	mux.HandleFunc("GET /api/v2/templates/{id}/versions/{n}", h.getVersion)
	mux.HandleFunc("PUT /api/v2/templates/{id}/versions/{n}/draft", h.saveDraft)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/publish", h.publish)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/docx-upload-url", h.presignDocxUpload)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/schema-upload-url", h.presignSchemaUpload)
	mux.HandleFunc("GET /api/v2/signed", h.signedDownload)
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor) {
		httpErr(w, 403, "forbidden")
		return
	}
	var req createTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, 400, "invalid_body")
		return
	}
	tenant := tenantFromRequest(r)
	actor := iamdomain.UserIDFromContext(r.Context())
	if actor == "" {
		actor = r.Header.Get("X-User-ID")
	}
	tpl, ver, err := h.svc.CreateTemplate(r.Context(), application.CreateTemplateCmd{
		TenantID: tenant, Key: req.Key, Name: req.Name, Description: req.Description, CreatedBy: actor,
	})
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 201, createTemplateResponse{ID: tpl.ID, VersionID: ver.ID})
}

const devTenantID = "00000000-0000-0000-0000-000000000001"

func tenantFromRequest(r *http.Request) string {
	if t := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); t != "" {
		return t
	}
	return devTenantID
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor, rolePublisher) {
		httpErr(w, 403, "forbidden")
		return
	}
	tenant := tenantFromRequest(r)
	tpls, err := h.svc.ListTemplates(r.Context(), tenant)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(tpls))
	for _, t := range tpls {
		out = append(out, map[string]any{
			"id":                   t.ID,
			"key":                  t.Key,
			"name":                 t.Name,
			"description":          t.Description,
			"latest_version":       t.LatestVersion,
			"latest_version_id":    t.LatestVersionID,
			"published_version_id": t.PublishedVersionID,
			"updated_at":           t.UpdatedAt,
		})
	}
	writeJSON(w, 200, out)
}

func (h *Handler) getVersion(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor, rolePublisher) {
		httpErr(w, 403, "forbidden")
		return
	}
	tplID := r.PathValue("id")
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		httpErr(w, 400, "invalid_version_num")
		return
	}
	tpl, ver, err := h.svc.GetVersion(r.Context(), tplID, n)
	if err != nil {
		httpErr(w, 404, "not_found")
		return
	}
	actor := r.Header.Get("X-User-ID")
	writeJSON(w, 200, map[string]any{
		"id": ver.ID, "template_id": tpl.ID, "name": tpl.Name,
		"version_num": ver.VersionNum, "status": string(ver.Status),
		"docx_storage_key": ver.DocxStorageKey, "schema_storage_key": ver.SchemaStorageKey,
		"lock_version": ver.LockVersion, "viewer_user_id": actor,
	})
}

func (h *Handler) saveDraft(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor) {
		httpErr(w, 403, "forbidden")
		return
	}
	var req saveDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, 400, "invalid_body")
		return
	}
	tplID := r.PathValue("id")
	n, convErr := strconv.Atoi(r.PathValue("n"))
	if convErr != nil {
		httpErr(w, 400, "invalid_version_num")
		return
	}
	_, ver, err := h.svc.GetVersion(r.Context(), tplID, n)
	if err != nil {
		httpErr(w, 404, "not_found")
		return
	}
	err = h.svc.SaveDraft(r.Context(), application.SaveDraftCmd{
		VersionID: ver.ID, ExpectedLockVersion: req.ExpectedLockVersion,
		DocxStorageKey: req.DocxStorageKey, SchemaStorageKey: req.SchemaStorageKey,
		DocxContentHash: req.DocxContentHash, SchemaContentHash: req.SchemaContentHash,
	})
	if errors.Is(err, domain.ErrLockVersionMismatch) {
		httpErr(w, 409, "template_draft_stale")
		return
	}
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, rolePublisher) {
		httpErr(w, 403, "forbidden")
		return
	}
	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, 400, "invalid_body")
		return
	}
	tplID := r.PathValue("id")
	n, convErr := strconv.Atoi(r.PathValue("n"))
	if convErr != nil {
		httpErr(w, 400, "invalid_version_num")
		return
	}
	_, ver, err := h.svc.GetVersion(r.Context(), tplID, n)
	if err != nil {
		httpErr(w, 404, "not_found")
		return
	}
	actor := r.Header.Get("X-User-ID")
	res, err := h.svc.PublishVersion(r.Context(), application.PublishCmd{
		VersionID: ver.ID, ActorUserID: actor, DocxKey: req.DocxKey, SchemaKey: req.SchemaKey,
	})
	var ve application.ValidationError
	if errors.As(err, &ve) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(422)
		_, _ = w.Write(ve.Raw)
		return
	}
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{
		"published_version_id":   ver.ID,
		"next_draft_id":          res.NewDraftID,
		"next_draft_version_num": res.NewDraftVersion,
	})
}

func (h *Handler) presignDocxUpload(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor) {
		httpErr(w, 403, "forbidden")
		return
	}
	tplID := r.PathValue("id")
	n, convErr := strconv.Atoi(r.PathValue("n"))
	if convErr != nil {
		httpErr(w, 400, "invalid_version_num")
		return
	}
	url, key, err := h.svc.PresignDocxUpload(r.Context(), tplID, n)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"url": url, "storage_key": key})
}

func (h *Handler) presignSchemaUpload(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor) {
		httpErr(w, 403, "forbidden")
		return
	}
	tplID := r.PathValue("id")
	n, convErr := strconv.Atoi(r.PathValue("n"))
	if convErr != nil {
		httpErr(w, 400, "invalid_version_num")
		return
	}
	url, key, err := h.svc.PresignSchemaUpload(r.Context(), tplID, n)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"url": url, "storage_key": key})
}

func (h *Handler) signedDownload(w http.ResponseWriter, r *http.Request) {
	if !requireRole(r, roleAdmin, roleTemplateAuthor, rolePublisher) {
		httpErr(w, 403, "forbidden")
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		httpErr(w, 400, "missing_key")
		return
	}
	if !strings.HasPrefix(key, "tenants/") || !strings.Contains(key, "/templates/") {
		httpErr(w, 403, "forbidden_key")
		return
	}
	url, err := h.svc.PresignObjectDownload(r.Context(), key)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
