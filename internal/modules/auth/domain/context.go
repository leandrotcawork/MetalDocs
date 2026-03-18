package domain

import "context"

type authContextKey string

const authPrincipalKey authContextKey = "auth.principal"

func WithCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, authPrincipalKey, user)
}

func CurrentUserFromContext(ctx context.Context) (CurrentUser, bool) {
	if ctx == nil {
		return CurrentUser{}, false
	}
	user, ok := ctx.Value(authPrincipalKey).(CurrentUser)
	return user, ok
}
