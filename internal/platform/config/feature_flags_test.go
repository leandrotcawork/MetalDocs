package config

import "testing"

func TestDocxV2Enabled_Default(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "")
	cfg := LoadFeatureFlagsConfig()
	if cfg.DocxV2Enabled {
		t.Fatalf("expected default false, got true")
	}
}

func TestDocxV2Enabled_True(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "true")
	cfg := LoadFeatureFlagsConfig()
	if !cfg.DocxV2Enabled {
		t.Fatalf("expected true")
	}
}

func TestDocxV2Enabled_False(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "false")
	cfg := LoadFeatureFlagsConfig()
	if cfg.DocxV2Enabled {
		t.Fatalf("expected false")
	}
}

func TestDocxV2Enabled_Unknown(t *testing.T) {
	t.Setenv("METALDOCS_DOCX_V2_ENABLED", "notabool")
	cfg := LoadFeatureFlagsConfig()
	if cfg.DocxV2Enabled {
		t.Fatalf("unknown must default to false")
	}
}
