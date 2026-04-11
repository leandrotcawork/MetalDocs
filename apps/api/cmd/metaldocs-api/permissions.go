package main

import (
	"net/http"
	"strings"

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

		if method == http.MethodPost && path == "/api/v1/telemetry/mddm-shadow-diff" {
			return iamdomain.PermDocumentRead, true
		}

		return "", false
	}
}
