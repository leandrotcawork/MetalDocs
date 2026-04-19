package featureflags_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/featureflags"
)

func TestHandler_Returns200WithFlags(t *testing.T) {
	h := featureflags.NewHandler(config.FeatureFlagsConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["MDDM_NATIVE_EXPORT_ROLLOUT_PCT"]; !ok {
		t.Fatal("expected MDDM_NATIVE_EXPORT_ROLLOUT_PCT in response")
	}
}
