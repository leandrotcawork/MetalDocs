package httpdelivery

import (
	"errors"
	"net/http"
	"strings"

	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type Middleware struct {
	service *authapp.Service
	cfg     authapp.Config
	enabled bool
}

func NewMiddleware(service *authapp.Service, cfg authapp.Config, enabled bool) *Middleware {
	return &Middleware{service: service, cfg: cfg, enabled: enabled}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	if !m.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path, r.Method) {
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

func isPublicPath(path, method string) bool {
	switch {
	case path == "/api/v1/health/live", path == "/api/v1/health/ready", path == "/api/v1/metrics":
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
