package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/security"
)

func TestRateLimiterBlocksWhenLimitExceeded(t *testing.T) {
	rl := security.NewRateLimiter(config.RateLimitConfig{
		Enabled:       true,
		WindowSeconds: 60,
		MaxRequests:   1,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := rl.Wrap(mux)

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req1.Header.Set("X-User-Id", "user-1")
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	req2.Header.Set("X-User-Id", "user-1")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d", rr2.Code)
	}
}

func TestRateLimiterIsolatedByIdentity(t *testing.T) {
	rl := security.NewRateLimiter(config.RateLimitConfig{
		Enabled:       true,
		WindowSeconds: 60,
		MaxRequests:   1,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/search/documents", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := rl.Wrap(mux)

	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/search/documents", nil)
	req1.Header.Set("X-User-Id", "user-a")
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/search/documents", nil)
	req2.Header.Set("X-User-Id", "user-b")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)

	if rr1.Code != http.StatusOK || rr2.Code != http.StatusOK {
		t.Fatalf("expected both users allowed, got user-a=%d user-b=%d", rr1.Code, rr2.Code)
	}
}

func TestRateLimiterSkipsHealthAndMetrics(t *testing.T) {
	rl := security.NewRateLimiter(config.RateLimitConfig{
		Enabled:       true,
		WindowSeconds: 60,
		MaxRequests:   1,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := rl.Wrap(mux)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected health 200, got %d", rr.Code)
		}
	}

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected metrics 200, got %d", rr.Code)
		}
	}
}
