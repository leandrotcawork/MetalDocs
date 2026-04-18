package main

import (
	"net/http"
	"strings"

	authdelivery "metaldocs/internal/modules/auth/delivery/http"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func newPermissionResolver() iamdelivery.PermissionResolver {
	return func(method, path string) (iamdomain.Permission, bool) {
		if path == "/api/v1/metrics" {
			return iamdomain.PermIAMManageRoles, true
		}
		if path == "/api/v1/health/live" || path == "/api/v1/health/ready" {
			return "", false
		}
		if strings.HasPrefix(path, "/api/v1/auth/") {
			return "", false
		}
		if method == http.MethodGet && path == "/api/v1/feature-flags" {
			return "", false
		}

		if method == http.MethodPost && path == "/api/v1/documents" {
			return iamdomain.PermDocumentCreate, true
		}
		if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/attachments") {
			return iamdomain.PermDocumentUploadAttachment, true
		}
		if method == http.MethodGet && path == "/api/v1/documents" {
			return iamdomain.PermDocumentRead, true
		}
		if method == http.MethodGet && path == "/api/v1/document-types" {
			return iamdomain.PermDocumentRead, true
		}
		if method == http.MethodGet && path == "/api/v1/document-templates" {
			return iamdomain.PermDocumentRead, true
		}
		if method == http.MethodGet && strings.HasPrefix(path, "/api/v1/documents/") && !strings.HasSuffix(path, "/versions") {
			return iamdomain.PermDocumentRead, true
		}
		if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/submit-for-approval") {
			return iamdomain.PermWorkflowTransition, true
		}
		if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/content") {
			return iamdomain.PermDocumentEdit, true
		}
		if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/content/browser") {
			return iamdomain.PermDocumentEdit, true
		}
		if method == http.MethodPut && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/template-assignment") {
			return iamdomain.PermDocumentEdit, true
		}
		if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/versions") {
			return iamdomain.PermDocumentEdit, true
		}
		if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/documents/") && strings.HasSuffix(path, "/export/docx") {
			return iamdomain.PermDocumentRead, true
		}
		if method == http.MethodGet && path == "/api/v1/search/documents" {
			return iamdomain.PermSearchRead, true
		}
		if method == http.MethodGet && path == "/api/v1/notifications" {
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
		if method == http.MethodGet && path == "/api/v1/iam/admin/overview" {
			return iamdomain.PermIAMManageRoles, true
		}

		// Template admin (Phase 2). Fine-grained RBAC is enforced inside the
		// service layer via isAllowedTemplate; at the HTTP boundary we only
		// need to require an authenticated session. PermTemplateView is the
		// least-privileged template permission and is granted to every role
		// that can access any template endpoint.
		if path == "/api/v1/templates" || strings.HasPrefix(path, "/api/v1/templates/") {
			return iamdomain.PermTemplateView, true
		}

		// docx-v2 templates (W2+): fine-grained per-method RBAC.
		if strings.HasPrefix(path, "/api/v2/templates") {
			switch {
			case method == http.MethodGet:
				return iamdomain.PermTemplateView, true
			case method == http.MethodPost && path == "/api/v2/templates":
				return iamdomain.PermTemplateEdit, true
			case method == http.MethodPut && strings.HasSuffix(path, "/draft"):
				return iamdomain.PermTemplateEdit, true
			case method == http.MethodPost && strings.HasSuffix(path, "/publish"):
				return iamdomain.PermTemplatePublish, true
			case method == http.MethodPost && strings.HasSuffix(path, "/docx-upload-url"):
				return iamdomain.PermTemplateEdit, true
			case method == http.MethodPost && strings.HasSuffix(path, "/schema-upload-url"):
				return iamdomain.PermTemplateEdit, true
			}
		}
		if strings.HasPrefix(path, "/api/v2/documents") {
			switch {
			case method == http.MethodGet:
				return iamdomain.PermDocumentRead, true
			case method == http.MethodPost && path == "/api/v2/documents":
				return iamdomain.PermDocumentCreate, true
			case method == http.MethodPost && strings.HasSuffix(path, "/finalize"):
				return iamdomain.PermWorkflowTransition, true
			case method == http.MethodPost && strings.HasSuffix(path, "/archive"):
				return iamdomain.PermDocumentEdit, true
			case method == http.MethodPost && strings.Contains(path, "/session/force-release"):
				return iamdomain.PermDocumentManagePermissions, true
			case method == http.MethodPost && strings.Contains(path, "/session/"):
				return iamdomain.PermDocumentEdit, true
			case method == http.MethodPost && strings.Contains(path, "/autosave/"):
				return iamdomain.PermDocumentEdit, true
			case method == http.MethodPost && strings.Contains(path, "/checkpoints/") && strings.HasSuffix(path, "/restore"):
				return iamdomain.PermDocumentEdit, true
			case method == http.MethodPost && strings.Contains(path, "/checkpoints"):
				return iamdomain.PermDocumentEdit, true
			case method == http.MethodPost && strings.HasSuffix(path, "/export/pdf"):
				return iamdomain.PermDocumentRead, true
			}
		}
		if method == http.MethodGet && path == "/api/v2/signed" {
			return iamdomain.PermTemplateView, true
		}

		return "", false
	}
}

// newPublicPathChecker derives a PublicPathChecker from the permission resolver.
//
// A route is public (no session cookie required) when the resolver says it is
// not guarded (guarded=false), UNLESS it is one of the session-required-but-
// unguarded endpoints below. That carve-out exists because "not guarded by
// IAM" (no RBAC permission required) is a different concern from "no session
// required". For example, GET /api/v1/auth/me and POST /api/v1/auth/change-
// password need a resolved session (so the handler knows which user) but are
// not gated by any IAM permission.
//
// The resolver remains the single source of truth for IAM perms; this checker
// only adds the minimal auth-scoped exceptions.
func newPublicPathChecker(resolver iamdelivery.PermissionResolver) authdelivery.PublicPathChecker {
	return func(method, path string) bool {
		if requiresSessionButNoPermission(method, path) {
			return false
		}
		_, guarded := resolver(method, path)
		return !guarded
	}
}

func requiresSessionButNoPermission(method, path string) bool {
	if method == http.MethodGet && path == "/api/v1/auth/me" {
		return true
	}
	if method == http.MethodPost && path == "/api/v1/auth/change-password" {
		return true
	}
	return false
}
