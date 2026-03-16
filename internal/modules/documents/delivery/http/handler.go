package httpdelivery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
)

type Handler struct {
	service *application.Service
}

type CreateDocumentRequest struct {
	Title          string `json:"title"`
	OwnerID        string `json:"ownerId"`
	Classification string `json:"classification"`
	InitialContent string `json:"initialContent,omitempty"`
}

type DocumentResponse struct {
	DocumentID string `json:"documentId"`
	Title      string `json:"title"`
	OwnerID    string `json:"ownerId"`
	Status     string `json:"status"`
}

type DocumentCreatedResponse struct {
	DocumentID string `json:"documentId"`
	Version    int    `json:"version"`
	Status     string `json:"status"`
}

type VersionResponse struct {
	DocumentID string `json:"documentId"`
	Version    int    `json:"version"`
	CreatedAt  string `json:"createdAt"`
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

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/health/live", h.handleHealthLive)
	mux.HandleFunc("/api/v1/health/ready", h.handleHealthReady)
	mux.HandleFunc("/api/v1/documents", h.handleDocuments)
	mux.HandleFunc("/api/v1/documents/", h.handleDocumentSubRoutes)
}

func (h *Handler) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "live"})
}

func (h *Handler) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) handleDocuments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateDocument(w, r)
	case http.MethodGet:
		h.handleListDocuments(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON payload", traceID)
		return
	}

	docID, err := newDocumentID()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate document id", traceID)
		return
	}

	doc, err := h.service.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:     docID,
		Title:          req.Title,
		OwnerID:        req.OwnerID,
		Classification: req.Classification,
		InitialContent: req.InitialContent,
		TraceID:        traceID,
	})
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	writeJSON(w, http.StatusCreated, DocumentCreatedResponse{
		DocumentID: doc.ID,
		Version:    1,
		Status:     doc.Status,
	})
}

func (h *Handler) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	docs, err := h.service.ListDocuments(context.Background())
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	out := make([]DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		out = append(out, DocumentResponse{
			DocumentID: doc.ID,
			Title:      doc.Title,
			OwnerID:    doc.OwnerID,
			Status:     doc.Status,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) handleDocumentSubRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/documents/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || parts[1] != "versions" {
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Route not found", requestTraceID(r))
		return
	}

	h.handleListVersions(w, r, parts[0])
}

func (h *Handler) handleListVersions(w http.ResponseWriter, r *http.Request, documentID string) {
	traceID := requestTraceID(r)

	versions, err := h.service.ListVersions(context.Background(), documentID)
	if err != nil {
		h.writeDomainError(w, err, traceID)
		return
	}

	items := make([]VersionResponse, 0, len(versions))
	for _, v := range versions {
		items = append(items, VersionResponse{
			DocumentID: v.DocumentID,
			Version:    v.Number,
			CreatedAt:  v.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error, traceID string) {
	switch {
	case errors.Is(err, domain.ErrInvalidCommand):
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request data", traceID)
	case errors.Is(err, domain.ErrDocumentNotFound):
		writeAPIError(w, http.StatusNotFound, "DOC_NOT_FOUND", "Document not found", traceID)
	case errors.Is(err, domain.ErrDocumentAlreadyExists):
		writeAPIError(w, http.StatusConflict, "CONFLICT_ERROR", "Document already exists", traceID)
	default:
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", traceID)
	}
}

func requestTraceID(r *http.Request) string {
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-Id")); traceID != "" {
		return traceID
	}
	return "trace-local"
}

func newDocumentID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
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
