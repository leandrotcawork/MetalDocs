package httpdelivery

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	searchapp "metaldocs/internal/modules/search/application"
	searchdomain "metaldocs/internal/modules/search/domain"
)

type Handler struct {
	service *searchapp.Service
}

type SearchDocumentResponse struct {
	DocumentID     string `json:"documentId"`
	Title          string `json:"title"`
	OwnerID        string `json:"ownerId"`
	Classification string `json:"classification"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
}

func NewHandler(service *searchapp.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/search/documents", h.handleSearchDocuments)
}

func (h *Handler) handleSearchDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid limit value", requestTraceID(r))
			return
		}
		limit = n
	}

	items, err := h.service.SearchDocuments(r.Context(), searchdomain.Query{
		Text:           r.URL.Query().Get("q"),
		OwnerID:        r.URL.Query().Get("ownerId"),
		Classification: r.URL.Query().Get("classification"),
		Status:         r.URL.Query().Get("status"),
		Limit:          limit,
	})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", requestTraceID(r))
		return
	}

	out := make([]SearchDocumentResponse, 0, len(items))
	for _, item := range items {
		out = append(out, SearchDocumentResponse{
			DocumentID:     item.ID,
			Title:          item.Title,
			OwnerID:        item.OwnerID,
			Classification: item.Classification,
			Status:         item.Status,
			CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
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
	writeJSON(w, status, apiErrorEnvelope{
		Error: apiError{
			Code:    code,
			Message: message,
			Details: map[string]any{},
			TraceID: traceID,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
