package config

import (
	"os"
	"strconv"
	"strings"
)

// DocgenV2Config holds connection settings for the docgen-v2 service.
type DocgenV2Config struct {
	Enabled               bool
	APIURL                string
	ServiceToken          string
	RequestTimeoutSeconds int
}

// LoadDocgenV2Config reads docgen-v2 config from environment variables.
// The service is considered disabled when METALDOCS_DOCGEN_V2_URL is empty.
func LoadDocgenV2Config() DocgenV2Config {
	apiURL := strings.TrimSpace(os.Getenv("METALDOCS_DOCGEN_V2_URL"))
	if apiURL == "" {
		return DocgenV2Config{}
	}
	token := strings.TrimSpace(os.Getenv("METALDOCS_DOCGEN_V2_SERVICE_TOKEN"))
	timeoutSeconds := 10
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_DOCGEN_V2_REQUEST_TIMEOUT_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}
	return DocgenV2Config{
		Enabled:               true,
		APIURL:                apiURL,
		ServiceToken:          token,
		RequestTimeoutSeconds: timeoutSeconds,
	}
}
