package config

import (
	"os"
	"strconv"
	"strings"
)

type CarboneConfig struct {
	Enabled               bool
	APIURL                string
	TemplateRoot          string
	RequestTimeoutSeconds int
}

func LoadCarboneConfig() CarboneConfig {
	apiURL := strings.TrimSpace(os.Getenv("METALDOCS_CARBONE_API_URL"))
	enabled := apiURL != ""

	templateRoot := strings.TrimSpace(os.Getenv("METALDOCS_CARBONE_TEMPLATE_ROOT"))
	if templateRoot == "" {
		templateRoot = "carbone/templates"
	}

	timeoutSeconds := 10
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_CARBONE_REQUEST_TIMEOUT_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			timeoutSeconds = parsed
		}
	}

	return CarboneConfig{
		Enabled:               enabled,
		APIURL:                apiURL,
		TemplateRoot:          templateRoot,
		RequestTimeoutSeconds: timeoutSeconds,
	}
}
