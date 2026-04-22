package observability

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct {
	provider RuntimeStatusProvider
}

func NewHealthHandler(provider RuntimeStatusProvider) *HealthHandler {
	return &HealthHandler{provider: provider}
}

func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/health/live", h.handleLive)
	mux.HandleFunc("/api/v1/health/ready", h.handleReady)
	mux.HandleFunc("/healthz", h.handleLive)
}

func (h *HealthHandler) handleLive(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "live", "checks": []map[string]any{{"name": "process", "status": "up"}}})
		return
	}
	status, payload := h.provider.Live(r.Context())
	writeJSON(w, status, payload)
}

func (h *HealthHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready", "checks": []map[string]any{{"name": "process", "status": "up"}}})
		return
	}
	status, payload := h.provider.Ready(r.Context())
	writeJSON(w, status, payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
