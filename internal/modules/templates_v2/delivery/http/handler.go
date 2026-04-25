package http

import (
	"net/http"
	"strings"

	iamdomain "metaldocs/internal/modules/iam/domain"
	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/platform/httpresponse"
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
	mux.HandleFunc("GET /api/v2/templates/{id}/versions/{n}/docx-url", h.getDocxURL)
	mux.HandleFunc("GET /api/v2/templates/{id}/audit", h.listAudit)
}

var (
	writeJSON = httpresponse.WriteJSON
	readJSON  = httpresponse.ReadJSON
)

func tenantIDFromReq(r *http.Request) string {
	if t := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); t != "" {
		return t
	}
	return devTenantID
}

func userIDFromReq(r *http.Request) string {
	return iamdomain.UserIDFromContext(r.Context())
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	httpresponse.WriteJSON(w, status, map[string]any{
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
