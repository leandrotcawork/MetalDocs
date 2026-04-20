package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"metaldocs/internal/modules/templates_v2/application"
)

const devTenantID = "00000000-0000-0000-0000-000000000001"

type AuthzFunc func(r *http.Request, tenantID, area string, action string) error

type Handler struct {
	svc   *application.Service
	authz AuthzFunc
}

func New(svc *application.Service, authz AuthzFunc) *Handler {
	if authz == nil {
		authz = func(*http.Request, string, string, string) error { return nil }
	}
	return &Handler{svc: svc, authz: authz}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v2/templates", h.createTemplate)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions", h.createNextVersion)
	mux.HandleFunc("PUT /api/v2/templates/{id}/versions/{n}/schema", h.updateSchemas)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/autosave/presign", h.presignAutosave)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/autosave/commit", h.commitAutosave)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/submit", h.submitForReview)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/review", h.review)
	mux.HandleFunc("POST /api/v2/templates/{id}/versions/{n}/approve", h.approve)
	mux.HandleFunc("POST /api/v2/templates/{id}/archive", h.archiveTemplate)
	mux.HandleFunc("PUT /api/v2/templates/{id}/approval-config", h.upsertApprovalConfig)

	mux.HandleFunc("GET /api/v2/templates", h.listTemplates)
	mux.HandleFunc("GET /api/v2/templates/{id}", h.getTemplate)
	mux.HandleFunc("GET /api/v2/templates/{id}/versions/{n}", h.getVersion)
	mux.HandleFunc("GET /api/v2/templates/{id}/audit", h.listAudit)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func tenantIDFromReq(r *http.Request) string {
	if t := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); t != "" {
		return t
	}
	return devTenantID
}

func userIDFromReq(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-User-ID"))
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeMappedErr(w http.ResponseWriter, err error) {
	status, code := MapErr(err)
	msg := err.Error()
	if msg == "" {
		msg = code
	}
	writeErr(w, status, code, msg)
}
