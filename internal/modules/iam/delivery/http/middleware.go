package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	authdomain "metaldocs/internal/modules/auth/domain"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type ctxKeyCapability struct{}
type ctxKeyAreaCode struct{}
type ctxKeyResourceID struct{}

type Middleware struct {
	authorizer   iamdomain.Authorizer
	roleProvider iamdomain.RoleProvider
	enabled      bool
	legacyHeader bool
	resolver     PermissionResolver
}

type PermissionResolver func(method, path string) (iamdomain.Permission, bool)

func NewMiddleware(authorizer iamdomain.Authorizer, roleProvider iamdomain.RoleProvider, enabled bool, legacyHeader ...bool) *Middleware {
	allowLegacy := false
	if len(legacyHeader) > 0 {
		allowLegacy = legacyHeader[0]
	}
	return &Middleware{
		authorizer:   authorizer,
		roleProvider: roleProvider,
		enabled:      enabled,
		legacyHeader: allowLegacy,
		resolver:     requiredPermission,
	}
}

// WithPermissionResolver allows composition root to define the route->permission mapping
// closer to where routes are registered, reducing the risk of drift.
func (m *Middleware) WithPermissionResolver(resolver PermissionResolver) *Middleware {
	if resolver == nil {
		return m
	}
	m.resolver = resolver
	return m
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	if !m.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolver := m.resolver
		if resolver == nil {
			resolver = requiredPermission
		}
		perm, guarded := resolver(r.Method, r.URL.Path)
		if !guarded {
			next.ServeHTTP(w, r)
			return
		}

		traceID := requestTraceID(r)
		userID := iamdomain.UserIDFromContext(r.Context())
		roles := iamdomain.RolesFromContext(r.Context())
		if userID == "" && m.legacyHeader {
			userID = strings.TrimSpace(r.Header.Get("X-User-Id"))
		}
		if userID == "" {
			writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
			return
		}

		if len(roles) == 0 {
			resolvedRoles, err := m.roleProvider.RolesByUserID(r.Context(), userID)
			if err != nil {
				if errors.Is(err, iamdomain.ErrUserNotFound) || errors.Is(err, iamdomain.ErrUserInactive) {
					writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "User is not authorized", traceID)
					return
				}
				writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authorization lookup failed", traceID)
				return
			}
			roles = resolvedRoles
		}

		if !hasPermission(m.authorizer, roles, perm) {
			writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Insufficient permissions", traceID)
			return
		}

		ctx := r.Context()
		if _, ok := authdomain.CurrentUserFromContext(ctx); !ok {
			ctx = iamdomain.WithAuthContext(ctx, userID, roles)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hasPermission(authorizer iamdomain.Authorizer, roles []iamdomain.Role, permission iamdomain.Permission) bool {
	for _, role := range roles {
		if authorizer.Can(role, permission) {
			return true
		}
	}
	return false
}

func requiredPermission(method, path string) (iamdomain.Permission, bool) {
	if path == "/api/v1/health/live" || path == "/api/v1/health/ready" || path == "/api/v1/metrics" {
		if path == "/api/v1/metrics" {
			return iamdomain.PermIAMManageRoles, true
		}
		return "", false
	}
	if strings.HasPrefix(path, "/api/v1/auth/") {
		return "", false
	}

	if method == http.MethodPost && path == "/api/v1/documents" {
		return iamdomain.PermDocumentCreate, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/attachments") {
		return iamdomain.PermDocumentUploadAttachment, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/collaboration/presence") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/collaboration/presence") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/collaboration/lock") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/collaboration/lock") {
		return iamdomain.PermDocumentEdit, true
	}
	if method == http.MethodDelete && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/collaboration/lock") {
		return iamdomain.PermDocumentEdit, true
	}
	if method == http.MethodGet && path == "/api/v1/documents" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && path == "/api/v1/document-types" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && path == "/api/v1/document-profiles" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/document-profiles/") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && path == "/api/v1/document-profiles" {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/document-profiles/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/document-profiles/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodDelete && strings.HasPrefix(path, "/api/v1/document-profiles/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodGet && path == "/api/v1/process-areas" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && path == "/api/v1/document-subjects" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && path == "/api/v1/process-areas" {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPost && path == "/api/v1/document-subjects" {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/process-areas/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodDelete && strings.HasPrefix(path, "/api/v1/process-areas/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/document-subjects/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodDelete && strings.HasPrefix(path, "/api/v1/document-subjects/") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && !strings.HasSuffix(path, "/versions") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/versions") {
		return iamdomain.PermDocumentEdit, true
	}
	if method == http.MethodGet && path == "/api/v1/search/documents" {
		return iamdomain.PermSearchRead, true
	}
	if method == http.MethodGet && path == "/api/v1/notifications" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && path == "/api/v1/operations/stream" {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/notifications/") && strings.HasSuffix(path, "/read") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && strings.Contains(path, "/versions/diff") {
		return iamdomain.PermVersionRead, true
	}
	if (method == http.MethodGet || method == http.MethodPut) && path == "/api/v1/access-policies" {
		return iamdomain.PermDocumentManagePermissions, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/versions") {
		return iamdomain.PermVersionRead, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/workflow/documents/") && strings.HasSuffix(path, "/transitions") {
		return iamdomain.PermWorkflowTransition, true
	}
	if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/workflow/documents/") && strings.HasSuffix(path, "/approvals") {
		return iamdomain.PermDocumentRead, true
	}
	if method == http.MethodPost && path == "/api/v1/iam/users" {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodGet && path == "/api/v1/iam/users" {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPatch && strings.HasPrefix(path, "/api/v1/iam/users/") && !strings.HasSuffix(path, "/roles") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/roles") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/roles") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/reset-password") {
		return iamdomain.PermIAMManageRoles, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/iam/users/") && strings.HasSuffix(path, "/unlock") {
		return iamdomain.PermIAMManageRoles, true
	}

	return "", false
}

type apiErrorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
	TraceID string         `json:"trace_id"`
}

func requestTraceID(r *http.Request) string {
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id")); traceID != "" {
		return traceID
	}
	return "trace-local"
}

func writeAPIError(w http.ResponseWriter, status int, code, message, traceID string) {
	writeJSON(w, status, apiErrorEnvelope{
		Error: apiError{
			Code:    code,
			Message: message,
			Details: map[string]any{},
			TraceID: traceID,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func NewV2AuthzMiddleware(service *iamapp.AuthorizationService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if service == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawCap := r.Context().Value(ctxKeyCapability{})
			if rawCap == nil {
				next.ServeHTTP(w, r)
				return
			}

			var capability iamdomain.Capability
			switch value := rawCap.(type) {
			case iamdomain.Capability:
				capability = value
			case string:
				capability = iamdomain.Capability(strings.TrimSpace(value))
			}
			if capability == "" {
				next.ServeHTTP(w, r)
				return
			}

			areaCode, _ := r.Context().Value(ctxKeyAreaCode{}).(string)
			areaCode = strings.TrimSpace(areaCode)
			if areaCode == "" {
				areaCode = strings.TrimSpace(r.Header.Get("X-Area-Code"))
			}

			resourceID, _ := r.Context().Value(ctxKeyResourceID{}).(string)
			userID := strings.TrimSpace(iamdomain.UserIDFromContext(r.Context()))
			if userID == "" {
				userID = strings.TrimSpace(r.Header.Get("X-User-Id"))
			}
			tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
			if tenantID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"code":                "missing_tenant",
					"required_capability": capability,
					"area_code":           areaCode,
				})
				return
			}

			err := service.Check(r.Context(), userID, tenantID, capability, iamapp.ResourceCtx{
				AreaCode:   areaCode,
				ResourceID: strings.TrimSpace(resourceID),
			})
			if err != nil {
				code := "forbidden"
				code = err.Error()
				writeJSON(w, http.StatusForbidden, map[string]any{
					"code":                code,
					"required_capability": capability,
					"area_code":           areaCode,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
