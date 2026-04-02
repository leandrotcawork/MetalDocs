package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/platform/config"
)

func TestBuildAPIDependenciesIncludesGotenbergCheckWhenHealthy(t *testing.T) {
	t.Setenv("METALDOCS_GOTENBERG_URL", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("METALDOCS_GOTENBERG_URL", server.URL)

	deps, err := BuildAPIDependencies(context.Background(), config.RepositoryMemory, config.AttachmentsConfig{
		Provider: config.StorageProviderMemory,
	})
	if err != nil {
		t.Fatalf("BuildAPIDependencies() error = %v", err)
	}

	statusCode, payload := deps.StatusProvider.Ready(context.Background())
	if statusCode != http.StatusOK {
		t.Fatalf("Ready() statusCode = %d, want %d", statusCode, http.StatusOK)
	}

	check := findCheck(t, payload, "gotenberg")
	if got := check["status"]; got != "up" {
		t.Fatalf("gotenberg status = %v, want up", got)
	}
	if got := check["detail"]; got != server.URL {
		t.Fatalf("gotenberg detail = %v, want %q", got, server.URL)
	}
}

func TestBuildAPIDependenciesMarksGotenbergSkippedWhenNotConfigured(t *testing.T) {
	t.Setenv("METALDOCS_GOTENBERG_URL", "")

	deps, err := BuildAPIDependencies(context.Background(), config.RepositoryMemory, config.AttachmentsConfig{
		Provider: config.StorageProviderMemory,
	})
	if err != nil {
		t.Fatalf("BuildAPIDependencies() error = %v", err)
	}

	statusCode, payload := deps.StatusProvider.Ready(context.Background())
	if statusCode != http.StatusOK {
		t.Fatalf("Ready() statusCode = %d, want %d", statusCode, http.StatusOK)
	}

	check := findCheck(t, payload, "gotenberg")
	if got := check["status"]; got != "skipped" {
		t.Fatalf("gotenberg status = %v, want skipped", got)
	}
	if got := check["detail"]; got != "gotenberg not configured" {
		t.Fatalf("gotenberg detail = %v, want %q", got, "gotenberg not configured")
	}
}

func TestBuildAPIDependenciesMarksGotenbergDownWhenHealthCheckFails(t *testing.T) {
	t.Setenv("METALDOCS_GOTENBERG_URL", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	t.Setenv("METALDOCS_GOTENBERG_URL", server.URL)

	deps, err := BuildAPIDependencies(context.Background(), config.RepositoryMemory, config.AttachmentsConfig{
		Provider: config.StorageProviderMemory,
	})
	if err != nil {
		t.Fatalf("BuildAPIDependencies() error = %v", err)
	}

	statusCode, payload := deps.StatusProvider.Ready(context.Background())
	if statusCode != http.StatusOK {
		t.Fatalf("Ready() statusCode = %d, want %d", statusCode, http.StatusOK)
	}

	check := findCheck(t, payload, "gotenberg")
	if got := check["status"]; got != "down" {
		t.Fatalf("gotenberg status = %v, want down", got)
	}
	if got := check["detail"]; got != "gotenberg unhealthy: status 503" {
		t.Fatalf("gotenberg detail = %v, want %q", got, "gotenberg unhealthy: status 503")
	}
}

func findCheck(t *testing.T, payload map[string]any, name string) map[string]any {
	t.Helper()

	checks, ok := payload["checks"].([]map[string]any)
	if !ok {
		t.Fatalf("checks payload type = %T, want []map[string]any", payload["checks"])
	}
	for _, check := range checks {
		if check["name"] == name {
			return check
		}
	}
	t.Fatalf("check %q not found in %#v", name, checks)
	return nil
}
