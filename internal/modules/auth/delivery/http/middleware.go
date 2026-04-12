package httpdelivery

import (
	"errors"
	"net/http"
	"strings"

	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

// PublicPathChecker returns true if the given method+path requires no session
// cookie (i.e. it is fully unauthenticated). Injecting this function into the
// middleware lets the composition root own the single authoritative list of
// public routes, preventing the auth layer and the IAM permission layer from
// maintaining two independent lists that can drift apart.
type PublicPathChecker func(method, path string) bool

type Middleware struct {
	service      *authapp.Service
	cfg          authapp.Config
	enabled      bool
	publicChecker PublicPathChecker // optional; falls back to defaultPublicPaths
}

func NewMiddleware(service *authapp.Service, cfg authapp.Config, enabled bool) *Middleware {
	return &Middleware{service: service, cfg: cfg, enabled: enabled}
}

// WithPublicPathChecker replaces the built-in public-path list with the
// provided checker. Use this in the composition root so there is one
// authoritative source of truth for which routes bypass authentication.
func (m *Middleware) WithPublicPathChecker(fn PublicPathChecker) *Middleware {
	m.publicChecker = fn
	return m
}

func (m *Middleware) isPublic(method, path string) bool {
	if m.publicChecker != nil {
		return m.publicChecker(method, path)
	}
	return defaultPublicPaths(method, path)
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	if !m.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.isPublic(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if m.cfg.LegacyHeaderEnabled && strings.TrimSpace(r.Header.Get("X-User-Id")) != "" {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(m.cfg.SessionCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", requestTraceID(r))
			return
		}

		currentUser, err := m.service.ResolveSession(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, authdomain.ErrSessionNotFound) || errors.Is(err, authdomain.ErrSessionExpired) || errors.Is(err, authdomain.ErrSessionRevoked) {
				writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", requestTraceID(r))
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authentication failed", requestTraceID(r))
			return
		}
		if currentUser.MustChangePassword && !isPasswordChangeAllowedPath(r.URL.Path, r.Method) {
			writeAPIError(w, http.StatusForbidden, "AUTH_PASSWORD_CHANGE_REQUIRED", "Password change is required before accessing the application", requestTraceID(r))
			return
		}

		ctx := authdomain.WithCurrentUser(r.Context(), currentUser)
		ctx = iamdomain.WithAuthContext(ctx, currentUser.UserID, currentUser.Roles)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// defaultPublicPaths is the fallback used when no PublicPathChecker is
// injected. Keep this in sync with the composition root's authoritative list
// whenever WithPublicPathChecker is not used (e.g. tests).
func defaultPublicPaths(method, path string) bool {
	switch {
	case path == "/api/v1/health/live", path == "/api/v1/health/ready":
		return true
	case method == http.MethodPost && path == "/api/v1/auth/login":
		return true
	case method == http.MethodPost && path == "/api/v1/auth/logout":
		return true
	default:
		return false
	}
}

func isPasswordChangeAllowedPath(path, method string) bool {
	return (method == http.MethodGet && path == "/api/v1/auth/me") ||
		(method == http.MethodPost && path == "/api/v1/auth/change-password") ||
		(method == http.MethodPost && path == "/api/v1/auth/logout")
}
