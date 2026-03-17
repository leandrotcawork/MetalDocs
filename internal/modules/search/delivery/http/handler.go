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
	DocumentID     string   `json:"documentId"`
	Title          string   `json:"title"`
	DocumentType   string   `json:"documentType"`
	OwnerID        string   `json:"ownerId"`
	BusinessUnit   string   `json:"businessUnit"`
	Department     string   `json:"department"`
	Classification string   `json:"classification"`
	Status         string   `json:"status"`
	Tags           []string `json:"tags"`
	EffectiveAt    string   `json:"effectiveAt,omitempty"`
	ExpiryAt       string   `json:"expiryAt,omitempty"`
	CreatedAt      string   `json:"createdAt"`
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

	expiryBefore, err := parseOptionalDateTimeQuery(r, "expiryBefore")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid expiryBefore value", requestTraceID(r))
		return
	}
	expiryAfter, err := parseOptionalDateTimeQuery(r, "expiryAfter")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid expiryAfter value", requestTraceID(r))
		return
	}

	items, err := h.service.SearchDocuments(r.Context(), searchdomain.Query{
		Text:           r.URL.Query().Get("q"),
		DocumentType:   r.URL.Query().Get("documentType"),
		OwnerID:        r.URL.Query().Get("ownerId"),
		BusinessUnit:   r.URL.Query().Get("businessUnit"),
		Department:     r.URL.Query().Get("department"),
		Classification: r.URL.Query().Get("classification"),
		Status:         r.URL.Query().Get("status"),
		Tag:            r.URL.Query().Get("tag"),
		ExpiryBefore:   expiryBefore,
		ExpiryAfter:    expiryAfter,
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
			DocumentType:   item.DocumentType,
			OwnerID:        item.OwnerID,
			BusinessUnit:   item.BusinessUnit,
			Department:     item.Department,
			Classification: item.Classification,
			Status:         item.Status,
			Tags:           append([]string(nil), item.Tags...),
			EffectiveAt:    formatOptionalTime(item.EffectiveAt),
			ExpiryAt:       formatOptionalTime(item.ExpiryAt),
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

func parseOptionalDateTimeQuery(r *http.Request, key string) (*time.Time, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	utc := parsed.UTC()
	return &utc, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
