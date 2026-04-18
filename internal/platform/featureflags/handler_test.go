package featureflags_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/featureflags"
)

func TestHandler_IncludesDocxV2Enabled(t *testing.T) {
	h := featureflags.NewHandler(config.FeatureFlagsConfig{DocxV2Enabled: true})
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
	if body["DOCX_V2_ENABLED"] != true {
		t.Fatalf("expected DOCX_V2_ENABLED=true in response, got %v", body["DOCX_V2_ENABLED"])
	}
}

func TestHandler_DocxV2Disabled_DefaultFalse(t *testing.T) {
	h := featureflags.NewHandler(config.FeatureFlagsConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/feature-flags", nil)
	rr := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)

	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["DOCX_V2_ENABLED"] != false {
		t.Fatalf("expected DOCX_V2_ENABLED=false, got %v", body["DOCX_V2_ENABLED"])
	}
}
