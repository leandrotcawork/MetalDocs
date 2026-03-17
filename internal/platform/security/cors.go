package security

import (
  "net/http"
  "strconv"
  "strings"

  "metaldocs/internal/platform/config"
)

type CORS struct {
  enabled          bool
  allowedOrigins   map[string]struct{}
  allowAllOrigins  bool
  allowedMethods   string
  allowedHeaders   string
  exposedHeaders   string
  allowCredentials bool
  maxAgeSeconds    int
}

func NewCORS(cfg config.CORSConfig) *CORS {
  origins := make(map[string]struct{}, len(cfg.AllowedOrigins))
  allowAll := false
  for _, origin := range cfg.AllowedOrigins {
    if origin == "*" {
      allowAll = true
      continue
    }
    origins[origin] = struct{}{}
  }

  return &CORS{
    enabled:          cfg.Enabled,
    allowedOrigins:   origins,
    allowAllOrigins:  allowAll,
    allowedMethods:   strings.Join(cfg.AllowedMethods, ", "),
    allowedHeaders:   strings.Join(cfg.AllowedHeaders, ", "),
    exposedHeaders:   strings.Join(cfg.ExposedHeaders, ", "),
    allowCredentials: cfg.AllowCredentials,
    maxAgeSeconds:    cfg.MaxAgeSeconds,
  }
}

func (c *CORS) Wrap(next http.Handler) http.Handler {
  if !c.enabled {
    return next
  }

  return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
    origin := strings.TrimSpace(req.Header.Get("Origin"))
    if origin == "" {
      next.ServeHTTP(w, req)
      return
    }

    if !c.isAllowedOrigin(origin) {
      if req.Method == http.MethodOptions {
        w.WriteHeader(http.StatusForbidden)
        return
      }
      next.ServeHTTP(w, req)
      return
    }

    setVary(w.Header(), "Origin")
    setVary(w.Header(), "Access-Control-Request-Method")
    setVary(w.Header(), "Access-Control-Request-Headers")

    if c.allowAllOrigins && !c.allowCredentials {
      w.Header().Set("Access-Control-Allow-Origin", "*")
    } else {
      w.Header().Set("Access-Control-Allow-Origin", origin)
    }

    if c.allowCredentials {
      w.Header().Set("Access-Control-Allow-Credentials", "true")
    }
    if c.exposedHeaders != "" {
      w.Header().Set("Access-Control-Expose-Headers", c.exposedHeaders)
    }

    if req.Method == http.MethodOptions {
      w.Header().Set("Access-Control-Allow-Methods", c.allowedMethods)
      w.Header().Set("Access-Control-Allow-Headers", c.allowedHeaders)
      w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.maxAgeSeconds))
      w.WriteHeader(http.StatusNoContent)
      return
    }

    next.ServeHTTP(w, req)
  })
}

func (c *CORS) isAllowedOrigin(origin string) bool {
  if c.allowAllOrigins {
    return true
  }
  _, ok := c.allowedOrigins[origin]
  return ok
}

func setVary(header http.Header, value string) {
  current := header.Values("Vary")
  for _, item := range current {
    for _, part := range strings.Split(item, ",") {
      if strings.EqualFold(strings.TrimSpace(part), value) {
        return
      }
    }
  }
  header.Add("Vary", value)
}
