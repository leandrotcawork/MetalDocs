package httpdelivery

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	auditdomain "metaldocs/internal/modules/audit/domain"
	authapp "metaldocs/internal/modules/auth/application"
	authdomain "metaldocs/internal/modules/auth/domain"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type AdminHandler struct {
	service     *iamapp.AdminService
	authService *authapp.Service
	audit       auditdomain.Writer
}

type UpsertUserRoleRequest struct {
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	AssignedBy  string `json:"assignedBy,omitempty"`
}

type ReplaceUserRolesRequest struct {
	DisplayName string   `json:"displayName"`
	Roles       []string `json:"roles"`
	AssignedBy  string   `json:"assignedBy,omitempty"`
}

type CreateUserRequest struct {
	UserID      string   `json:"userId,omitempty"`
	Username    string   `json:"username"`
	Email       string   `json:"email,omitempty"`
	DisplayName string   `json:"displayName"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles"`
}

type UpdateUserRequest struct {
	DisplayName        *string `json:"displayName,omitempty"`
	Email              *string `json:"email,omitempty"`
	IsActive           *bool   `json:"isActive,omitempty"`
	NewPassword        string  `json:"newPassword,omitempty"`
	MustChangePassword *bool   `json:"mustChangePassword,omitempty"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

func NewAdminHandler(service *iamapp.AdminService, authService *authapp.Service, auditWriter ...auditdomain.Writer) *AdminHandler {
	var writer auditdomain.Writer
	if len(auditWriter) > 0 {
		writer = auditWriter[0]
	}
	return &AdminHandler{service: service, authService: authService, audit: writer}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/iam/users", h.handleUsers)
	mux.HandleFunc("/api/v1/iam/users/", h.handleUserRoute)
}

func (h *AdminHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleListUsers(w, r)
	case http.MethodPost:
		h.handleCreateUser(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) handleUserRoute(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/iam/users/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "roles" {
		switch r.Method {
		case http.MethodPost:
			h.handleUserRoleUpsert(w, r, strings.TrimSpace(parts[0]), traceID)
		case http.MethodPut:
			h.handleReplaceUserRoles(w, r, strings.TrimSpace(parts[0]), traceID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "reset-password" && r.Method == http.MethodPost {
		h.handleResetPassword(w, r, strings.TrimSpace(parts[0]), traceID)
		return
	}
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "unlock" && r.Method == http.MethodPost {
		h.handleUnlockUser(w, r, strings.TrimSpace(parts[0]), traceID)
		return
	}
	if len(parts) == 1 && strings.TrimSpace(parts[0]) != "" && r.Method == http.MethodPatch {
		h.handlePatchUser(w, r, strings.TrimSpace(parts[0]), traceID)
		return
	}
	writeAPIError(w, http.StatusNotFound, "INTERNAL_ERROR", "Route not found", traceID)
}

func (h *AdminHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	if h.authService == nil {
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "User management service is not configured", traceID)
		return
	}
	items, err := h.authService.ListUsers(r.Context())
	if err != nil {
		log.Printf("iam admin: list users failed: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users", traceID)
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		roles := make([]string, 0, len(item.Roles))
		for _, role := range item.Roles {
			roles = append(roles, string(role))
		}
		out = append(out, map[string]any{
			"userId":              item.UserID,
			"username":            item.Username,
			"email":               item.Email,
			"displayName":         item.DisplayName,
			"isActive":            item.IsActive,
			"mustChangePassword":  item.MustChangePassword,
			"failedLoginAttempts": item.FailedLoginAttempts,
			"roles":               roles,
			"lastLoginAt":         formatOptionalTime(item.LastLoginAt),
			"lockedUntil":         formatOptionalTime(item.LockedUntil),
			"createdAt":           item.CreatedAt.UTC().Format(time.RFC3339),
			"updatedAt":           item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *AdminHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)
	if h.authService == nil {
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "User management service is not configured", traceID)
		return
	}
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	roles, ok := parseRoles(req.Roles)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid roles", traceID)
		return
	}
	assignedBy := authenticatedActor(r)
	if err := h.authService.CreateUser(r.Context(), req.UserID, req.Username, req.Email, req.DisplayName, req.Password, roles, assignedBy); err != nil {
		h.writeAuthError(w, err, traceID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"userId": strings.TrimSpace(defaultString(req.UserID, req.Username))})
}

func (h *AdminHandler) handlePatchUser(w http.ResponseWriter, r *http.Request, userID, traceID string) {
	if h.authService == nil {
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "User management service is not configured", traceID)
		return
	}
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	if err := h.authService.UpdateUser(r.Context(), authdomain.UpdateUserParams{
		UserID:             userID,
		DisplayName:        req.DisplayName,
		Email:              req.Email,
		IsActive:           req.IsActive,
		MustChangePassword: req.MustChangePassword,
	}, req.NewPassword); err != nil {
		h.writeAuthError(w, err, traceID)
		return
	}
	h.recordAudit(r, userID, "iam.user.updated", map[string]any{
		"displayName":        req.DisplayName,
		"email":              req.Email,
		"isActive":           req.IsActive,
		"mustChangePassword": req.MustChangePassword,
	})
	writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "updated": true})
}

func (h *AdminHandler) handleUserRoleUpsert(w http.ResponseWriter, r *http.Request, userID, traceID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req UpsertUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	role := iamdomain.Role(strings.ToLower(strings.TrimSpace(req.Role)))
	if role == "" {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Role is required", traceID)
		return
	}

	switch role {
	case iamdomain.RoleAdmin, iamdomain.RoleEditor, iamdomain.RoleReviewer, iamdomain.RoleViewer:
	default:
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role", traceID)
		return
	}

	assignedBy := strings.TrimSpace(req.AssignedBy)
	if assignedBy == "" {
		assignedBy = authenticatedActor(r)
	}

	if err := h.service.UpsertUserAndAssignRole(r.Context(), userID, req.DisplayName, role, assignedBy); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to upsert user role", traceID)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"userId":      userID,
		"role":        string(role),
		"displayName": strings.TrimSpace(req.DisplayName),
	})
}

func (h *AdminHandler) handleReplaceUserRoles(w http.ResponseWriter, r *http.Request, userID, traceID string) {
	var req ReplaceUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	roles, ok := parseRoles(req.Roles)
	if !ok {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid roles", traceID)
		return
	}

	assignedBy := strings.TrimSpace(req.AssignedBy)
	if assignedBy == "" {
		assignedBy = authenticatedActor(r)
	}

	if err := h.service.ReplaceUserRoles(r.Context(), userID, req.DisplayName, roles, assignedBy); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to replace user roles", traceID)
		return
	}

	roleStrings := make([]string, 0, len(roles))
	for _, role := range roles {
		roleStrings = append(roleStrings, string(role))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":      userID,
		"displayName": strings.TrimSpace(req.DisplayName),
		"roles":       roleStrings,
	})
	h.recordAudit(r, userID, "iam.user.roles.replaced", map[string]any{
		"roles": roleStrings,
	})
}

func (h *AdminHandler) handleResetPassword(w http.ResponseWriter, r *http.Request, userID, traceID string) {
	if h.authService == nil {
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "User management service is not configured", traceID)
		return
	}
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}
	if err := h.authService.AdminResetPassword(r.Context(), userID, req.NewPassword); err != nil {
		h.writeAuthError(w, err, traceID)
		return
	}
	h.recordAudit(r, userID, "auth.user.password_reset", map[string]any{
		"mustChangePassword": true,
	})
	writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "reset": true, "mustChangePassword": true})
}

func (h *AdminHandler) handleUnlockUser(w http.ResponseWriter, r *http.Request, userID, traceID string) {
	if h.authService == nil {
		writeAPIError(w, http.StatusNotImplemented, "INTERNAL_ERROR", "User management service is not configured", traceID)
		return
	}
	if err := h.authService.UnlockUser(r.Context(), userID); err != nil {
		h.writeAuthError(w, err, traceID)
		return
	}
	h.recordAudit(r, userID, "auth.user.unlocked", map[string]any{})
	writeJSON(w, http.StatusOK, map[string]any{"userId": userID, "unlocked": true})
}

func (h *AdminHandler) writeAuthError(w http.ResponseWriter, err error, traceID string) {
	switch {
	case errors.Is(err, authdomain.ErrPasswordPolicy):
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
	case errors.Is(err, authdomain.ErrUserAlreadyExists):
		writeAPIError(w, http.StatusConflict, "CONFLICT_ERROR", "User already exists", traceID)
	case errors.Is(err, authdomain.ErrIdentityNotFound):
		writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "User not found", traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process user request", traceID)
	}
}

func (h *AdminHandler) recordAudit(r *http.Request, userID, action string, payload map[string]any) {
	if h.audit == nil {
		return
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = h.audit.Record(r.Context(), auditdomain.Event{
		ID:           "evt_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", ""),
		OccurredAt:   time.Now().UTC(),
		ActorID:      authenticatedActor(r),
		Action:       action,
		ResourceType: "user",
		ResourceID:   userID,
		PayloadJSON:  string(payloadJSON),
		TraceID:      requestTraceID(r),
	})
}

func parseRoles(items []string) ([]iamdomain.Role, bool) {
	out := make([]iamdomain.Role, 0, len(items))
	seen := map[iamdomain.Role]bool{}
	for _, item := range items {
		role := iamdomain.Role(strings.ToLower(strings.TrimSpace(item)))
		switch role {
		case iamdomain.RoleAdmin, iamdomain.RoleEditor, iamdomain.RoleReviewer, iamdomain.RoleViewer:
		default:
			return nil, false
		}
		if !seen[role] {
			seen[role] = true
			out = append(out, role)
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func authenticatedActor(r *http.Request) string {
	if user, ok := authdomain.CurrentUserFromContext(r.Context()); ok && strings.TrimSpace(user.UserID) != "" {
		return user.UserID
	}
	return "system"
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
