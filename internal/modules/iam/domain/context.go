package domain

import "context"

type authContextKey string

const (
	authUserIDKey authContextKey = "iam.user_id"
	authRolesKey  authContextKey = "iam.roles"
)

func WithAuthContext(ctx context.Context, userID string, roles []Role) context.Context {
	ctx = context.WithValue(ctx, authUserIDKey, userID)
	return context.WithValue(ctx, authRolesKey, roles)
}

func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	userID, _ := ctx.Value(authUserIDKey).(string)
	return userID
}

func RolesFromContext(ctx context.Context) []Role {
	if ctx == nil {
		return nil
	}
	roles, _ := ctx.Value(authRolesKey).([]Role)
	return roles
}
