package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"testing"

	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestMiddlewareStripsUserIDHeaderAfterAuthContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req.Header.Set("X-User-ID", "attacker")
	req = req.WithContext(iamdomain.WithAuthContext(req.Context(), "real-user", []iamdomain.Role{iamdomain.RoleAdmin}))

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := iamdomain.UserIDFromContext(r.Context()); got != "real-user" {
			t.Fatalf("UserIDFromContext() = %q, want %q", got, "real-user")
		}
		if got := r.Header.Get("X-User-ID"); got != "" {
			t.Fatalf("X-User-ID header = %q, want empty", got)
		}
	})

	middleware := NewMiddleware(iamapp.NewStaticAuthorizer(), nil, true)
	middleware.Wrap(next).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler was not called")
	}
}
