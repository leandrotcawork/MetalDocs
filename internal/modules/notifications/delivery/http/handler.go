package httpdelivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	iamdomain "metaldocs/internal/modules/iam/domain"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
)

type Handler struct {
	service *notificationapp.Service
}

type NotificationResponse struct {
	ID              string `json:"id"`
	RecipientUserID string `json:"recipientUserId"`
	EventType       string `json:"eventType"`
	ResourceType    string `json:"resourceType"`
	ResourceID      string `json:"resourceId"`
	Title           string `json:"title"`
	Message         string `json:"message"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`
	ReadAt          string `json:"readAt,omitempty"`
}

type MarkReadResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	ReadAt string `json:"readAt"`
}

func NewHandler(service *notificationapp.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/notifications", h.handleNotifications)
	mux.HandleFunc("/api/v1/notifications/", h.handleNotificationRoute)
}

func (h *Handler) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID := iamdomain.UserIDFromContext(r.Context())
	roles := iamdomain.RolesFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", requestTraceID(r))
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid limit value", requestTraceID(r))
			return
		}
		limit = parsed
	}

	recipientUserID := userID
	if requestedRecipient := strings.TrimSpace(r.URL.Query().Get("recipientUserId")); requestedRecipient != "" {
		if !hasAdminRole(roles) && requestedRecipient != userID {
			writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Insufficient permissions", requestTraceID(r))
			return
		}
		recipientUserID = requestedRecipient
	}

	items, err := h.service.ListNotifications(r.Context(), notificationdomain.ListNotificationsQuery{
		RecipientUserID: recipientUserID,
		Status:          strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:           limit,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list notifications", requestTraceID(r))
		return
	}

	out := make([]NotificationResponse, 0, len(items))
	for _, item := range items {
		responseItem := NotificationResponse{
			ID:              item.ID,
			RecipientUserID: item.RecipientUserID,
			EventType:       item.EventType,
			ResourceType:    item.ResourceType,
			ResourceID:      item.ResourceID,
			Title:           item.Title,
			Message:         item.Message,
			Status:          item.Status,
			CreatedAt:       item.CreatedAt.UTC().Format(time.RFC3339),
		}
		if item.ReadAt != nil {
			responseItem.ReadAt = item.ReadAt.UTC().Format(time.RFC3339)
		}
		out = append(out, responseItem)
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleNotificationRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "read" {
		writeAPIError(w, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Route not found", requestTraceID(r))
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	userID := iamdomain.UserIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", requestTraceID(r))
		return
	}

	readAt := time.Now().UTC()
	if err := h.service.MarkNotificationRead(r.Context(), parts[0], userID); err != nil {
		if errors.Is(err, notificationapp.ErrNotificationNotFound) {
			writeAPIError(w, http.StatusNotFound, "NOTIFICATION_NOT_FOUND", "Notification not found", requestTraceID(r))
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update notification", requestTraceID(r))
		return
	}

	writeJSON(w, http.StatusOK, MarkReadResponse{
		ID:     parts[0],
		Status: notificationdomain.StatusRead,
		ReadAt: readAt.Format(time.RFC3339),
	})
}

func hasAdminRole(roles []iamdomain.Role) bool {
	for _, role := range roles {
		if role == iamdomain.RoleAdmin {
			return true
		}
	}
	return false
}

func requestTraceID(r *http.Request) string {
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id")); traceID != "" {
		return traceID
	}
	return "trace-local"
}

func writeAPIError(w http.ResponseWriter, status int, code, message, traceID string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":     code,
			"message":  message,
			"details":  map[string]any{},
			"trace_id": traceID,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
