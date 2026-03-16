package unit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	iamapp "metaldocs/internal/modules/iam/application"
	iamdelivery "metaldocs/internal/modules/iam/delivery/http"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type fakeRoleProvider struct {
	roles []iamdomain.Role
	err   error
}

func (f fakeRoleProvider) RolesByUserID(context.Context, string) ([]iamdomain.Role, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.roles, nil
}

func TestMiddlewareBlocksProtectedRouteWithoutUserID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddlewareAllowsWithRoleFromProvider(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req.Header.Set("X-User-Id", "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMiddlewareBlocksInsufficientPermission(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(`{"title":"x","ownerId":"y"}`))
	req.Header.Set("X-User-Id", "user-2")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestMiddlewareSkipsHealthRoutes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMiddlewareUnauthorizedWhenUserMissingInProvider(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{err: iamdomain.ErrUserNotFound}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req.Header.Set("X-User-Id", "missing-user")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddlewareInternalErrorWhenProviderFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{err: errors.New("db timeout")}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req.Header.Set("X-User-Id", "user-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestMiddlewareProtectsIAMAdminRoute(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/iam/users/test-user/roles", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleEditor}}, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users/test-user/roles", strings.NewReader(`{"role":"viewer"}`))
	req.Header.Set("X-User-Id", "editor-user")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
