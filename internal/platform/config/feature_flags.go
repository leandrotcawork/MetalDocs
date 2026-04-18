package config

import (
	"os"
	"strconv"
	"strings"
)

// FeatureFlagsConfig holds server-controlled feature flag values read from
// environment variables at startup.
type FeatureFlagsConfig struct {
	// MDDMNativeExportRolloutPercent is the percentage (0–100) of users for
	// whom the client-side MDDM DOCX export path is active.
	// Env: METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT (default 0)
	MDDMNativeExportRolloutPercent int
}

// LoadFeatureFlagsConfig reads feature flag config from environment variables.
func LoadFeatureFlagsConfig() FeatureFlagsConfig {
	pct := 0
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_MDDM_NATIVE_EXPORT_ROLLOUT_PCT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			if parsed < 0 {
				parsed = 0
			} else if parsed > 100 {
				parsed = 100
			}
			pct = parsed
		}
	}
	return FeatureFlagsConfig{
		MDDMNativeExportRolloutPercent: pct,
	}
}

func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "true" || v == "1" || v == "yes"
}
