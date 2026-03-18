package httpdelivery

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
)

type Handler struct {
	service *authapp.Service
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func NewHandler(service *authapp.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/login", h.handleLogin)
	mux.HandleFunc("/api/v1/auth/logout", h.handleLogout)
	mux.HandleFunc("/api/v1/auth/me", h.handleMe)
	mux.HandleFunc("/api/v1/auth/change-password", h.handleChangePassword)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	traceID := requestTraceID(r)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	session, err := h.service.Authenticate(r.Context(), req.Identifier, req.Password, r)
	if err != nil {
		log.Printf("auth login failed for %q: %v", strings.TrimSpace(req.Identifier), err)
		http.SetCookie(w, h.service.ExpiredSessionCookie())
		h.writeAuthError(w, err, traceID)
		return
	}
	http.SetCookie(w, h.service.SessionCookie(session.RawToken, session.ExpiresAt))
	writeJSON(w, http.StatusOK, map[string]any{
		"user":      session.CurrentUser,
		"expiresAt": session.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if cookie, err := r.Cookie(h.service.SessionCookieName()); err == nil {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, h.service.ExpiredSessionCookie())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	traceID := requestTraceID(r)
	user, ok := authdomain.CurrentUserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	traceID := requestTraceID(r)
	user, ok := authdomain.CurrentUserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	if err := h.service.ChangePasswordForUser(r.Context(), user, req.CurrentPassword, req.NewPassword); err != nil {
		log.Printf("auth change password failed for %q: %v", strings.TrimSpace(user.UserID), err)
		h.writeAuthError(w, err, traceID)
		return
	}
	currentUser, err := h.service.CurrentUser(r.Context(), user.UserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"changed": true,
		"user":    currentUser,
	})
}

func (h *Handler) writeAuthError(w http.ResponseWriter, err error, traceID string) {
	switch {
	case errors.Is(err, authdomain.ErrInvalidCredentials):
		writeAPIError(w, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Invalid username/email or password", traceID)
	case errors.Is(err, authdomain.ErrIdentityNotFound):
		writeAPIError(w, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Invalid username/email or password", traceID)
	case errors.Is(err, authdomain.ErrIdentityLocked):
		writeAPIError(w, http.StatusForbidden, "AUTH_ACCOUNT_LOCKED", "Account is temporarily locked", traceID)
	case errors.Is(err, authdomain.ErrPasswordPolicy):
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
	case errors.Is(err, authdomain.ErrIdentityInactive):
		writeAPIError(w, http.StatusForbidden, "AUTH_ACCOUNT_INACTIVE", "User account is inactive", traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
	}
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
	writeJSON(w, status, apiErrorEnvelope{Error: apiError{Code: code, Message: message, Details: map[string]any{}, TraceID: traceID}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
