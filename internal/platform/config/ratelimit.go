package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type RateLimitConfig struct {
	Enabled       bool
	WindowSeconds int
	MaxRequests   int
}

func LoadRateLimitConfig() (RateLimitConfig, error) {
	enabled := strings.EqualFold(strings.TrimSpace(os.Getenv("METALDOCS_RATE_LIMIT_ENABLED")), "true")

	window := 60
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_RATE_LIMIT_WINDOW_SECONDS")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return RateLimitConfig{}, fmt.Errorf("invalid METALDOCS_RATE_LIMIT_WINDOW_SECONDS: %s", raw)
		}
		window = n
	}

	maxReq := 120
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_RATE_LIMIT_MAX_REQUESTS")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return RateLimitConfig{}, fmt.Errorf("invalid METALDOCS_RATE_LIMIT_MAX_REQUESTS: %s", raw)
		}
		maxReq = n
	}

	return RateLimitConfig{
		Enabled:       enabled,
		WindowSeconds: window,
		MaxRequests:   maxReq,
	}, nil
}
