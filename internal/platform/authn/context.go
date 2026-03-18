package authn

import (
	"context"
	"strings"

	iamdomain "metaldocs/internal/modules/iam/domain"
)

// UserIDFromContext resolves authenticated user identity from request context.
func UserIDFromContext(ctx context.Context) string {
	return strings.TrimSpace(iamdomain.UserIDFromContext(ctx))
}

// RolesFromContext resolves authenticated roles as normalized lowercase strings.
func RolesFromContext(ctx context.Context) []string {
	roles := iamdomain.RolesFromContext(ctx)
	if len(roles) == 0 {
		return nil
	}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		normalized := strings.ToLower(strings.TrimSpace(string(role)))
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
