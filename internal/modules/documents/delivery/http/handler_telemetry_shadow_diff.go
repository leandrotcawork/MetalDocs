package httpdelivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

// ShadowDiffRepo is the persistence port for shadow diff telemetry events.
type ShadowDiffRepo interface {
	Insert(ctx context.Context, event domain.ShadowDiffEvent) error
}

// ShadowDiffHandler handles POST /telemetry/mddm-shadow-diff.
type ShadowDiffHandler struct {
	repo ShadowDiffRepo
}

// NewShadowDiffHandler constructs a ShadowDiffHandler.
func NewShadowDiffHandler(repo ShadowDiffRepo) *ShadowDiffHandler {
	return &ShadowDiffHandler{repo: repo}
}

type shadowDiffRequest struct {
	DocumentID        string         `json:"document_id"`
	VersionNumber     int            `json:"version_number"`
	UserIDHash        string         `json:"user_id_hash"`
	CurrentXMLHash    string         `json:"current_xml_hash"`
	ShadowXMLHash     string         `json:"shadow_xml_hash"`
	DiffSummary       map[string]any `json:"diff_summary"`
	CurrentDurationMs int            `json:"current_duration_ms"`
	ShadowDurationMs  int            `json:"shadow_duration_ms"`
	ShadowError       string         `json:"shadow_error,omitempty"`
}

// Handle processes the incoming shadow diff telemetry request.
func (h *ShadowDiffHandler) Handle(w http.ResponseWriter, r *http.Request) {
	traceID := requestTraceID(r)

	if userIDFromContext(r.Context()) == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
		return
	}

	if h.repo == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "TELEMETRY_UNAVAILABLE", "Shadow diff telemetry is not configured", traceID)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB hard cap
	var req shadowDiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("decode body: %v", err), traceID)
		return
	}

	if req.DocumentID == "" || req.VersionNumber <= 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "document_id and version_number required", traceID)
		return
	}

	// Reject negative durations regardless of success/failure.
	if req.CurrentDurationMs < 0 || req.ShadowDurationMs < 0 {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "duration fields must be non-negative", traceID)
		return
	}

	// On success rows (no shadow_error), both hashes must be present.
	if req.ShadowError == "" && (req.CurrentXMLHash == "" || req.ShadowXMLHash == "") {
		writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "current_xml_hash and shadow_xml_hash required when shadow_error is empty", traceID)
		return
	}

	event := domain.ShadowDiffEvent{
		DocumentID:        req.DocumentID,
		VersionNumber:     req.VersionNumber,
		UserIDHash:        req.UserIDHash,
		CurrentXMLHash:    req.CurrentXMLHash,
		ShadowXMLHash:     req.ShadowXMLHash,
		DiffSummary:       req.DiffSummary,
		CurrentDurationMs: req.CurrentDurationMs,
		ShadowDurationMs:  req.ShadowDurationMs,
		ShadowError:       req.ShadowError,
		RecordedAt:        time.Now().UTC(),
		TraceID:           traceID,
	}

	if err := h.repo.Insert(r.Context(), event); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("insert: %v", err), traceID)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
