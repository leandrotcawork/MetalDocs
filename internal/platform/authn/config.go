package authn

import (
	"os"
	"strconv"
	"strings"
	"time"

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
