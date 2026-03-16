package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strings"

	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type AdminHandler struct {
	service *iamapp.AdminService
}

type UpsertUserRoleRequest struct {
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	AssignedBy  string `json:"assignedBy,omitempty"`
}

func NewAdminHandler(service *iamapp.AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/iam/users/", h.handleUserRoleUpsert)
}

func (h *AdminHandler) handleUserRoleUpsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	traceID := requestTraceID(r)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/iam/users/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "roles" {
		writeAPIError(w, http.StatusNotFound, "INTERNAL_ERROR", "Route not found", traceID)
		return
	}
	userID := strings.TrimSpace(parts[0])

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
		assignedBy = strings.TrimSpace(r.Header.Get("X-User-Id"))
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
