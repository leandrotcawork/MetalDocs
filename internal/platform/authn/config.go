package authn

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	authapp "metaldocs/internal/modules/auth/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func Enabled() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("METALDOCS_AUTH_ENABLED")))
	if raw == "" {
		return true
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func CacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("METALDOCS_AUTHZ_CACHE_TTL_SECONDS"))
	if raw == "" {
		return 30 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func LoadRuntimeConfig() (authapp.Config, error) {
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "local"
	}

	sessionSecret := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_SESSION_SECRET"))
	if sessionSecret == "" && Enabled() {
		return authapp.Config{}, fmt.Errorf("METALDOCS_AUTH_SESSION_SECRET is required when auth is enabled")
	}

	sessionCookieName := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_SESSION_COOKIE_NAME"))
	if sessionCookieName == "" {
		sessionCookieName = "metaldocs_session"
	}

	sessionTTLHours := 12
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_SESSION_TTL_HOURS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return authapp.Config{}, fmt.Errorf("invalid METALDOCS_AUTH_SESSION_TTL_HOURS")
		}
		sessionTTLHours = parsed
	}

	passwordMinLength := 8
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_PASSWORD_MIN_LENGTH")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 8 {
			return authapp.Config{}, fmt.Errorf("invalid METALDOCS_AUTH_PASSWORD_MIN_LENGTH")
		}
		passwordMinLength = parsed
	}

	maxFailedAttempts := 5
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_LOGIN_MAX_FAILED_ATTEMPTS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 3 {
			return authapp.Config{}, fmt.Errorf("invalid METALDOCS_AUTH_LOGIN_MAX_FAILED_ATTEMPTS")
		}
		maxFailedAttempts = parsed
	}

	lockMinutes := 15
	if raw := strings.TrimSpace(os.Getenv("METALDOCS_AUTH_LOGIN_LOCK_MINUTES")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return authapp.Config{}, fmt.Errorf("invalid METALDOCS_AUTH_LOGIN_LOCK_MINUTES")
		}
		lockMinutes = parsed
	}

	bootstrapEnabled := parseBoolEnv("METALDOCS_BOOTSTRAP_ADMIN_ENABLED", appEnv == "local")
	bootstrapUserID := strings.TrimSpace(os.Getenv("METALDOCS_BOOTSTRAP_ADMIN_USER_ID"))
	if bootstrapUserID == "" {
		bootstrapUserID = "admin-local"
	}
	bootstrapUsername := strings.TrimSpace(os.Getenv("METALDOCS_BOOTSTRAP_ADMIN_USERNAME"))
	if bootstrapUsername == "" {
		bootstrapUsername = "admin"
	}
	bootstrapName := strings.TrimSpace(os.Getenv("METALDOCS_BOOTSTRAP_ADMIN_DISPLAY_NAME"))
	if bootstrapName == "" {
		bootstrapName = "Administrator"
	}

	cfg := authapp.Config{
		SessionCookieName:      sessionCookieName,
		SessionTTL:             time.Duration(sessionTTLHours) * time.Hour,
		SessionSecret:          sessionSecret,
		PasswordMinLength:      passwordMinLength,
		LoginMaxFailedAttempts: maxFailedAttempts,
		LoginLockDuration:      time.Duration(lockMinutes) * time.Minute,
		LegacyHeaderEnabled:    parseBoolEnv("METALDOCS_AUTH_LEGACY_HEADER_ENABLED", false),
		BootstrapAdminEnabled:  bootstrapEnabled,
		BootstrapAdminUserID:   bootstrapUserID,
		BootstrapAdminUsername: bootstrapUsername,
		BootstrapAdminEmail:    strings.TrimSpace(os.Getenv("METALDOCS_BOOTSTRAP_ADMIN_EMAIL")),
		BootstrapAdminPassword: os.Getenv("METALDOCS_BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapAdminName:     bootstrapName,
		CookieSecure:           parseBoolEnv("METALDOCS_AUTH_COOKIE_SECURE", appEnv != "local"),
		TrustedOrigins:         splitCSV(os.Getenv("METALDOCS_AUTH_TRUSTED_ORIGINS")),
		OriginProtection:       parseBoolEnv("METALDOCS_AUTH_ORIGIN_PROTECTION_ENABLED", Enabled()),
	}

	if cfg.BootstrapAdminEnabled && strings.TrimSpace(cfg.BootstrapAdminPassword) == "" {
		return authapp.Config{}, fmt.Errorf("METALDOCS_BOOTSTRAP_ADMIN_PASSWORD is required when bootstrap admin is enabled")
	}

	return cfg, nil
}

// DevRoleMap parses METALDOCS_DEV_USER_ROLES as: user1:admin,user2:viewer|reviewer
func DevRoleMap() map[string][]iamdomain.Role {
	raw := strings.TrimSpace(os.Getenv("METALDOCS_DEV_USER_ROLES"))
	if raw == "" {
		return map[string][]iamdomain.Role{"admin-local": {iamdomain.RoleAdmin}}
	}

	out := map[string][]iamdomain.Role{}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		userID := strings.TrimSpace(parts[0])
		rolesRaw := strings.TrimSpace(parts[1])
		if userID == "" || rolesRaw == "" {
			continue
		}
		roleParts := strings.Split(rolesRaw, "|")
		roles := make([]iamdomain.Role, 0, len(roleParts))
		for _, rp := range roleParts {
			r := strings.TrimSpace(rp)
			if r == "" {
				continue
			}
			roles = append(roles, iamdomain.Role(r))
		}
		if len(roles) > 0 {
			out[userID] = roles
		}
	}
	if len(out) == 0 {
		out["admin-local"] = []iamdomain.Role{iamdomain.RoleAdmin}
	}
	return out
}

func parseBoolEnv(name string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}
	raw = strings.ToLower(raw)
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
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
