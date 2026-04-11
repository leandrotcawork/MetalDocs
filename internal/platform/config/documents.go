package config

import (
	"os"
	"strings"
)

// DocumentsConfig holds runtime configuration for the documents module.
type DocumentsConfig struct {
	// RendererVersion is the semver of the MDDM renderer currently running.
	// Pinned into released versions so export can load the exact renderer
	// snapshot used at release time. Defaults to "1.0.0" if unset.
	RendererVersion string

	// LayoutIRHash is the SHA-256 hash of the compiled Layout IR bundle.
	// Captured alongside RendererVersion to detect silent asset drift.
	// Empty string is acceptable before Task 10 wires hash computation.
	LayoutIRHash string
}

// LoadDocumentsConfig reads documents module config from environment variables.
func LoadDocumentsConfig() DocumentsConfig {
	rendererVersion := strings.TrimSpace(os.Getenv("METALDOCS_RENDERER_VERSION"))
	if rendererVersion == "" {
		rendererVersion = "1.0.0"
	}
	layoutIRHash := strings.TrimSpace(os.Getenv("METALDOCS_LAYOUT_IR_HASH"))
	return DocumentsConfig{
		RendererVersion: rendererVersion,
		LayoutIRHash:    layoutIRHash,
	}
}
