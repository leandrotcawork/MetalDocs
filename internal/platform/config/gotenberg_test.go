package config

import "testing"

func TestLoadGotenbergConfigDisabledWhenURLMissing(t *testing.T) {
	t.Setenv("METALDOCS_GOTENBERG_URL", "")

	cfg := LoadGotenbergConfig()

	if cfg.Enabled {
		t.Fatalf("expected disabled config when env var is empty")
	}
	if cfg.URL != "" {
		t.Fatalf("expected empty URL, got %q", cfg.URL)
	}
}

func TestLoadGotenbergConfigEnabledWhenURLPresent(t *testing.T) {
	t.Setenv("METALDOCS_GOTENBERG_URL", "http://localhost:3000")

	cfg := LoadGotenbergConfig()

	if !cfg.Enabled {
		t.Fatalf("expected enabled config when env var is set")
	}
	if cfg.URL != "http://localhost:3000" {
		t.Fatalf("expected URL %q, got %q", "http://localhost:3000", cfg.URL)
	}
}
