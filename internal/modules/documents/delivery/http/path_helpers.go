package httpdelivery

import (
	"context"
	"strings"

	iamdomain "metaldocs/internal/modules/iam/domain"
)

func userIDFromContext(ctx context.Context) string {
	return strings.TrimSpace(iamdomain.UserIDFromContext(ctx))
}

func extractDocIDFromPath(path string) string {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	for i := 0; i < len(parts); i++ {
		if parts[i] != "documents" {
			continue
		}
		if i+1 < len(parts) {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}
