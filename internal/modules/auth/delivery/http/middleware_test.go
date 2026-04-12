package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authapp "metaldocs/internal/modules/auth/application"
)

// passthrough is a sentinel handler that records whether it was reached.
func passthrough(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestDefaultPublicPaths(t *testing.T) {
	cases := []struct {
		method string
		path   string
		public bool
	}{
		{http.MethodGet, "/api/v1/health/live", true},
		{http.MethodGet, "/api/v1/health/ready", true},
		{http.MethodPost, "/api/v1/auth/login", true},
		{http.MethodPost, "/api/v1/auth/logout", true},
		// Non-public routes must NOT be exempt
		{http.MethodGet, "/api/v1/feature-flags", false},
		{http.MethodGet, "/api/v1/documents", false},
		{http.MethodPost, "/api/v1/documents", false},
		{http.MethodGet, "/api/v1/auth/me", false},
	}
	for _, tc := range cases {
		got := defaultPublicPaths(tc.method, tc.path)
		if got != tc.public {
			t.Errorf("defaultPublicPaths(%q, %q) = %v, want %v", tc.method, tc.path, got, tc.public)
		}
	}
}

func TestMiddleware_PublicPathChecker_Injection(t *testing.T) {
	// A custom checker that marks /api/v1/feature-flags as public.
	customChecker := func(method, path string) bool {
		if method == http.MethodGet && path == "/api/v1/feature-flags" {
			return true
		}
		return defaultPublicPaths(method, path)
	}

	m := NewMiddleware(nil, authapp.Config{}, true).
		WithPublicPathChecker(customChecker)

	reached := false
	handler := m.Wrap(passthrough(&reached))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for public feature-flags, got %d", rec.Code)
	}
	if !reached {
		t.Error("passthrough handler was not reached for public path")
	}
}

func TestMiddleware_NoCookie_PrivateRoute_Returns401(t *testing.T) {
	m := NewMiddleware(nil, authapp.Config{SessionCookieName: "session"}, true)

	reached := false
	handler := m.Wrap(passthrough(&reached))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated private route, got %d", rec.Code)
	}
	if reached {
		t.Error("passthrough handler must NOT be reached for unauthenticated private route")
	}
}

func TestMiddleware_Disabled_PassesThrough(t *testing.T) {
	m := NewMiddleware(nil, authapp.Config{}, false)

	reached := false
	handler := m.Wrap(passthrough(&reached))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !reached {
		t.Error("disabled middleware must pass all requests through")
	}
}

func TestMiddleware_DefaultPublicPaths_NoChecker(t *testing.T) {
	m := NewMiddleware(nil, authapp.Config{}, true) // no WithPublicPathChecker

	cases := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/v1/health/live", http.StatusOK},
		{http.MethodPost, "/api/v1/auth/login", http.StatusOK},
		// feature-flags is NOT in the default list — should be 401 without a checker
		{http.MethodGet, "/api/v1/feature-flags", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		reached := false
		handler := m.Wrap(passthrough(&reached))
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}
