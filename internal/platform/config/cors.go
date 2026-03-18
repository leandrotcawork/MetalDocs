package config

import (
  "fmt"
  "os"
  "strings"
)

type CORSConfig struct {
  Enabled          bool
  AllowedOrigins   []string
  AllowedMethods   []string
  AllowedHeaders   []string
  ExposedHeaders   []string
  AllowCredentials bool
  MaxAgeSeconds    int
}

func LoadCORSConfig() (CORSConfig, error) {
  enabled := strings.EqualFold(strings.TrimSpace(os.Getenv("METALDOCS_CORS_ENABLED")), "true")

  allowedOrigins := splitCSV(os.Getenv("METALDOCS_CORS_ALLOWED_ORIGINS"))
  allowedMethods := splitCSV(os.Getenv("METALDOCS_CORS_ALLOWED_METHODS"))
  if len(allowedMethods) == 0 {
    allowedMethods = []string{"GET", "POST", "PUT", "OPTIONS"}
  }
  allowedHeaders := splitCSV(os.Getenv("METALDOCS_CORS_ALLOWED_HEADERS"))
  if len(allowedHeaders) == 0 {
    allowedHeaders = []string{"Content-Type", "X-Trace-Id"}
  }
  exposedHeaders := splitCSV(os.Getenv("METALDOCS_CORS_EXPOSED_HEADERS"))
  allowCredentials := strings.EqualFold(strings.TrimSpace(os.Getenv("METALDOCS_CORS_ALLOW_CREDENTIALS")), "true")

  maxAge := 300
  if raw := strings.TrimSpace(os.Getenv("METALDOCS_CORS_MAX_AGE_SECONDS")); raw != "" {
    var parsed int
    _, err := fmt.Sscanf(raw, "%d", &parsed)
    if err != nil || parsed < 0 {
      return CORSConfig{}, fmt.Errorf("invalid METALDOCS_CORS_MAX_AGE_SECONDS: %s", raw)
    }
    maxAge = parsed
  }

  if allowCredentials {
    for _, origin := range allowedOrigins {
      if origin == "*" {
        return CORSConfig{}, fmt.Errorf("METALDOCS_CORS_ALLOWED_ORIGINS cannot contain * when credentials are enabled")
      }
    }
  }

  return CORSConfig{
    Enabled:          enabled,
    AllowedOrigins:   allowedOrigins,
    AllowedMethods:   normalizeUpper(allowedMethods),
    AllowedHeaders:   allowedHeaders,
    ExposedHeaders:   exposedHeaders,
    AllowCredentials: allowCredentials,
    MaxAgeSeconds:    maxAge,
  }, nil
}

func splitCSV(raw string) []string {
  if strings.TrimSpace(raw) == "" {
    return nil
  }
  parts := strings.Split(raw, ",")
  items := make([]string, 0, len(parts))
  for _, part := range parts {
    value := strings.TrimSpace(part)
    if value != "" {
      items = append(items, value)
    }
  }
  return items
}

func normalizeUpper(values []string) []string {
  normalized := make([]string, 0, len(values))
  for _, value := range values {
    normalized = append(normalized, strings.ToUpper(strings.TrimSpace(value)))
  }
  return normalized
}
