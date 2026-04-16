package main

import (
	"net/http"
	"testing"

	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestPermissionResolver(t *testing.T) {
	t.Parallel()

	resolver := newPermissionResolver()

	testCases := []struct {
		name       string
		method     string
		path       string
		wantPerm   iamdomain.Permission
		wantGuard  bool
	}{
		{name: "health live unguarded", method: http.MethodGet, path: "/api/v1/health/live", wantPerm: "", wantGuard: false},
		{name: "auth login unguarded", method: http.MethodPost, path: "/api/v1/auth/login", wantPerm: "", wantGuard: false},
		{name: "documents create", method: http.MethodPost, path: "/api/v1/documents", wantPerm: iamdomain.PermDocumentCreate, wantGuard: true},
		{name: "documents list", method: http.MethodGet, path: "/api/v1/documents", wantPerm: iamdomain.PermDocumentRead, wantGuard: true},
		{name: "document detail", method: http.MethodGet, path: "/api/v1/documents/doc-1", wantPerm: iamdomain.PermDocumentRead, wantGuard: true},
		{name: "document browser content save", method: http.MethodPost, path: "/api/v1/documents/doc-1/content/browser", wantPerm: iamdomain.PermDocumentEdit, wantGuard: true},
		{name: "document versions list", method: http.MethodGet, path: "/api/v1/documents/doc-1/versions", wantPerm: iamdomain.PermVersionRead, wantGuard: true},
		{name: "document version create", method: http.MethodPost, path: "/api/v1/documents/doc-1/versions", wantPerm: iamdomain.PermDocumentEdit, wantGuard: true},
		{name: "workflow transition", method: http.MethodPost, path: "/api/v1/workflow/documents/doc-1/transitions", wantPerm: iamdomain.PermWorkflowTransition, wantGuard: true},
		{name: "iam users list", method: http.MethodGet, path: "/api/v1/iam/users", wantPerm: iamdomain.PermIAMManageRoles, wantGuard: true},
		{name: "iam roles update", method: http.MethodPut, path: "/api/v1/iam/users/u-1/roles", wantPerm: iamdomain.PermIAMManageRoles, wantGuard: true},
		// feature-flags is fully public: no permission required, no session required.
		// This must remain unguarded so initFeatureFlags() can call it before login.
		{name: "feature flags unguarded", method: http.MethodGet, path: "/api/v1/feature-flags", wantPerm: "", wantGuard: false},
		{name: "unknown endpoint unguarded", method: http.MethodGet, path: "/api/v1/unknown", wantPerm: "", wantGuard: false},
		// Template admin (Phase 2) — all HTTP methods on /api/v1/templates[/...] require
		// an authenticated session; fine-grained RBAC is enforced inside the service layer.
		{name: "template list", method: http.MethodGet, path: "/api/v1/templates", wantPerm: iamdomain.PermTemplateView, wantGuard: true},
		{name: "template create", method: http.MethodPost, path: "/api/v1/templates", wantPerm: iamdomain.PermTemplateView, wantGuard: true},
		{name: "template draft sub-route", method: http.MethodGet, path: "/api/v1/templates/my-key/draft", wantPerm: iamdomain.PermTemplateView, wantGuard: true},
		{name: "template publish sub-route", method: http.MethodPost, path: "/api/v1/templates/my-key/publish", wantPerm: iamdomain.PermTemplateView, wantGuard: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotPerm, gotGuard := resolver(tc.method, tc.path)
			if gotGuard != tc.wantGuard {
				t.Fatalf("guard mismatch: got %v want %v", gotGuard, tc.wantGuard)
			}
			if gotPerm != tc.wantPerm {
				t.Fatalf("permission mismatch: got %q want %q", gotPerm, tc.wantPerm)
			}
		})
	}
}
