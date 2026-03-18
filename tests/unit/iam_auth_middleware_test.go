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

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true, true)
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

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true, true)
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

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true, true)
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

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMiddlewareRequiresAuthenticationForMetrics(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: []iamdomain.Role{iamdomain.RoleViewer}}, true, true)
	h := mw.Wrap(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMiddlewareUnauthorizedWhenUserMissingInProvider(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{err: iamdomain.ErrUserNotFound}, true, true)
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

	mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{err: errors.New("db timeout")}, true, true)
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
	tests := []struct {
		name       string
		roles      []iamdomain.Role
		userID     string
		wantStatus int
	}{
		{
			name:       "viewer is forbidden to manage roles",
			roles:      []iamdomain.Role{iamdomain.RoleViewer},
			userID:     "viewer-user",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "editor is forbidden to manage roles",
			roles:      []iamdomain.Role{iamdomain.RoleEditor},
			userID:     "editor-user",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "reviewer is forbidden to manage roles",
			roles:      []iamdomain.Role{iamdomain.RoleReviewer},
			userID:     "reviewer-user",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can manage roles",
			roles:      []iamdomain.Role{iamdomain.RoleAdmin},
			userID:     "admin-local",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/iam/users/test-user/roles", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: tt.roles}, true, true)
			h := mw.Wrap(mux)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/iam/users/test-user/roles", strings.NewReader(`{"role":"viewer"}`))
			req.Header.Set("X-User-Id", tt.userID)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestMiddlewareProtectsWorkflowTransitionRoute(t *testing.T) {
	tests := []struct {
		name       string
		roles      []iamdomain.Role
		userID     string
		wantStatus int
	}{
		{
			name:       "viewer is forbidden to transition workflow",
			roles:      []iamdomain.Role{iamdomain.RoleViewer},
			userID:     "viewer-user",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "reviewer can transition workflow",
			roles:      []iamdomain.Role{iamdomain.RoleReviewer},
			userID:     "reviewer-user",
			wantStatus: http.StatusOK,
		},
		{
			name:       "admin can transition workflow",
			roles:      []iamdomain.Role{iamdomain.RoleAdmin},
			userID:     "admin-local",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/workflow/documents/test-user/transitions", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: tt.roles}, true, true)
			h := mw.Wrap(mux)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflow/documents/test-user/transitions", strings.NewReader(`{"toStatus":"IN_REVIEW"}`))
			req.Header.Set("X-User-Id", tt.userID)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestMiddlewareProtectsSearchRoute(t *testing.T) {
	tests := []struct {
		name       string
		roles      []iamdomain.Role
		userID     string
		wantStatus int
	}{
		{
			name:       "viewer can search documents",
			roles:      []iamdomain.Role{iamdomain.RoleViewer},
			userID:     "viewer-user",
			wantStatus: http.StatusOK,
		},
		{
			name:       "reviewer can search documents",
			roles:      []iamdomain.Role{iamdomain.RoleReviewer},
			userID:     "reviewer-user",
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing user header is unauthorized",
			roles:      []iamdomain.Role{iamdomain.RoleViewer},
			userID:     "",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/search/documents", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: tt.roles}, true, true)
			h := mw.Wrap(mux)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search/documents?q=doc", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-Id", tt.userID)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestMiddlewareProtectsAccessPoliciesRoute(t *testing.T) {
	tests := []struct {
		name       string
		roles      []iamdomain.Role
		userID     string
		method     string
		wantStatus int
	}{
		{
			name:       "viewer is forbidden to list policies",
			roles:      []iamdomain.Role{iamdomain.RoleViewer},
			userID:     "viewer-user",
			method:     http.MethodGet,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "editor is forbidden to replace policies",
			roles:      []iamdomain.Role{iamdomain.RoleEditor},
			userID:     "editor-user",
			method:     http.MethodPut,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin can manage policies",
			roles:      []iamdomain.Role{iamdomain.RoleAdmin},
			userID:     "admin-local",
			method:     http.MethodPut,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v1/access-policies", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			mw := iamdelivery.NewMiddleware(iamapp.NewStaticAuthorizer(), fakeRoleProvider{roles: tt.roles}, true, true)
			h := mw.Wrap(mux)

			req := httptest.NewRequest(tt.method, "/api/v1/access-policies?resourceScope=document&resourceId=doc-1", strings.NewReader(`{"resourceScope":"document","resourceId":"doc-1","policies":[]}`))
			if tt.userID != "" {
				req.Header.Set("X-User-Id", tt.userID)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}
