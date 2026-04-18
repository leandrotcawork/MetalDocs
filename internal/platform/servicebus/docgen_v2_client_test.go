package servicebus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/platform/servicebus"
)

func TestDocgenV2Client_Health_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Service-Token"); got != "" {
			t.Fatalf("/health must NOT require token; got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"test"}`))
	}))
	defer srv.Close()

	c := servicebus.NewDocgenV2Client(srv.URL, "shh", 2*time.Second)
	ver, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "test" {
		t.Fatalf("expected version 'test', got %q", ver)
	}
}

func TestDocgenV2Client_Health_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := servicebus.NewDocgenV2Client(srv.URL, "shh", 500*time.Millisecond)
	if _, err := c.Health(context.Background()); err == nil {
		t.Fatal("expected error on 502")
	}
}
