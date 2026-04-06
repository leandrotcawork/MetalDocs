package config

import (
	"os"
	"strconv"
	"strings"
)

type DocgenConfig struct {
	Enabled               bool
	APIURL                string
	RequestTimeoutSeconds int
}

func LoadDocgenConfig() DocgenConfig {
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	apiURL := strings.TrimSpace(os.Getenv("METALDOCS_DOCGEN_API_URL"))
	if apiURL == "" && strings.EqualFold(appEnv, "local") {
		apiURL = "http://127.0.0.1:3001"
	}
	enabled := apiURL != ""

	timeoutSeconds := 10
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_DOCGEN_REQUEST_TIMEOUT_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}

	return DocgenConfig{
		Enabled:               enabled,
		APIURL:                apiURL,
		RequestTimeoutSeconds: timeoutSeconds,
	}
}
