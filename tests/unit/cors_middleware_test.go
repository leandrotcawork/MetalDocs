package unit

import (
  "net/http"
  "net/http/httptest"
  "testing"

  "metaldocs/internal/platform/config"
  "metaldocs/internal/platform/security"
)

func TestCORSPreflightAllowedOrigin(t *testing.T) {
  middleware := security.NewCORS(config.CORSConfig{
    Enabled:        true,
    AllowedOrigins: []string{"http://127.0.0.1:4173"},
    AllowedMethods: []string{"GET", "POST", "PUT", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type", "X-Trace-Id"},
    MaxAgeSeconds:  300,
  })

  req := httptest.NewRequest(http.MethodOptions, "/api/v1/documents", nil)
  req.Header.Set("Origin", "http://127.0.0.1:4173")
  req.Header.Set("Access-Control-Request-Method", http.MethodGet)
  rec := httptest.NewRecorder()

  middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    t.Fatalf("preflight should not reach next handler")
  })).ServeHTTP(rec, req)

  if rec.Code != http.StatusNoContent {
    t.Fatalf("expected 204, got %d", rec.Code)
  }
  if rec.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:4173" {
    t.Fatalf("unexpected allow origin header: %s", rec.Header().Get("Access-Control-Allow-Origin"))
  }
}

func TestCORSDeniesUnknownOriginPreflight(t *testing.T) {
  middleware := security.NewCORS(config.CORSConfig{
    Enabled:        true,
    AllowedOrigins: []string{"http://127.0.0.1:4173"},
    AllowedMethods: []string{"GET", "POST", "PUT", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type", "X-Trace-Id"},
    MaxAgeSeconds:  300,
  })

  req := httptest.NewRequest(http.MethodOptions, "/api/v1/documents", nil)
  req.Header.Set("Origin", "http://malicious.local")
  req.Header.Set("Access-Control-Request-Method", http.MethodGet)
  rec := httptest.NewRecorder()

  middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    t.Fatalf("denied preflight should not reach next handler")
  })).ServeHTTP(rec, req)

  if rec.Code != http.StatusForbidden {
    t.Fatalf("expected 403, got %d", rec.Code)
  }
}

func TestCORSAddsHeadersToSimpleRequest(t *testing.T) {
  middleware := security.NewCORS(config.CORSConfig{
    Enabled:        true,
    AllowedOrigins: []string{"http://127.0.0.1:4173"},
    AllowedMethods: []string{"GET", "POST", "PUT", "OPTIONS"},
    AllowedHeaders: []string{"Content-Type", "X-Trace-Id"},
    MaxAgeSeconds:  300,
  })

  req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
  req.Header.Set("Origin", "http://127.0.0.1:4173")
  rec := httptest.NewRecorder()

  middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
  })).ServeHTTP(rec, req)

  if rec.Code != http.StatusOK {
    t.Fatalf("expected 200, got %d", rec.Code)
  }
  if rec.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:4173" {
    t.Fatalf("unexpected allow origin header: %s", rec.Header().Get("Access-Control-Allow-Origin"))
  }
}
