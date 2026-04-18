// Package featureflags exposes server-controlled feature flag values to the
// frontend via GET /api/v1/feature-flags.
package featureflags

import (
	"encoding/json"
	"net/http"

	"metaldocs/internal/platform/config"
)

// Handler serves GET /api/v1/feature-flags.
type Handler struct {
	cfg config.FeatureFlagsConfig
}

// NewHandler creates a Handler backed by the given config.
func NewHandler(cfg config.FeatureFlagsConfig) *Handler {
	return &Handler{cfg: cfg}
}

// RegisterRoutes registers the feature-flags route on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/feature-flags", h.handle)
}

type featureFlagsResponse struct {
	MDDMNativeExportRolloutPct int `json:"MDDM_NATIVE_EXPORT_ROLLOUT_PCT"`
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(featureFlagsResponse{
		MDDMNativeExportRolloutPct: h.cfg.MDDMNativeExportRolloutPercent,
	})
}
