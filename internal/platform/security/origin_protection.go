package security

import (
	"net/http"
	"net/url"
	"strings"
)

type OriginProtectionConfig struct {
	Enabled           bool
	SessionCookieName string
	TrustedOrigins    []string
}

type OriginProtection struct {
	enabled        bool
	cookieName     string
	trustedOrigins map[string]struct{}
}

func NewOriginProtection(cfg OriginProtectionConfig) *OriginProtection {
	trusted := make(map[string]struct{}, len(cfg.TrustedOrigins))
	for _, origin := range cfg.TrustedOrigins {
		normalized := normalizeOrigin(origin)
		if normalized != "" {
			trusted[normalized] = struct{}{}
		}
	}

	return &OriginProtection{
		enabled:        cfg.Enabled,
		cookieName:     strings.TrimSpace(cfg.SessionCookieName),
		trustedOrigins: trusted,
	}
}

func (p *OriginProtection) Wrap(next http.Handler) http.Handler {
	if !p.enabled || p.cookieName == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresOriginProtection(r) {
			next.ServeHTTP(w, r)
			return
		}

		if _, err := r.Cookie(p.cookieName); err != nil {
			next.ServeHTTP(w, r)
			return
		}

		if origin := normalizeOrigin(r.Header.Get("Origin")); origin != "" {
			if p.isAllowedOrigin(r, origin) {
				next.ServeHTTP(w, r)
				return
			}
			writeOriginError(w)
			return
		}

		if refererOrigin := originFromReferer(r.Header.Get("Referer")); refererOrigin != "" {
			if p.isAllowedOrigin(r, refererOrigin) {
				next.ServeHTTP(w, r)
				return
			}
			writeOriginError(w)
			return
		}

		writeOriginError(w)
	})
}

func requiresOriginProtection(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (p *OriginProtection) isAllowedOrigin(r *http.Request, origin string) bool {
	if origin == sameOrigin(r) {
		return true
	}
	_, ok := p.trustedOrigins[origin]
	return ok
}

func sameOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	scheme := "http"
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.ToLower(forwardedProto)
	} else if r.TLS != nil {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		return ""
	}
	return normalizeOrigin(scheme + "://" + host)
}

func originFromReferer(referer string) string {
	ref := strings.TrimSpace(referer)
	if ref == "" {
		return ""
	}
	parsed, err := url.Parse(ref)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return normalizeOrigin(parsed.Scheme + "://" + parsed.Host)
}

func normalizeOrigin(origin string) string {
	value := strings.TrimSpace(origin)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host)
}

func writeOriginError(w http.ResponseWriter) {
	writeAPIError(w, http.StatusForbidden, "AUTH_INVALID_ORIGIN", "Cross-site session request blocked", "trace-local")
}
