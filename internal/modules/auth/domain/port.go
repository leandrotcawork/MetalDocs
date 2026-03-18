package domain

import (
	"context"
	"time"
)

type Repository interface {
	FindIdentityByIdentifier(ctx context.Context, identifier string) (Identity, error)
	FindIdentityByUserID(ctx context.Context, userID string) (Identity, error)
	CreateSession(ctx context.Context, session Session) error
	FindSession(ctx context.Context, sessionID string) (Session, error)
	TouchSession(ctx context.Context, sessionID string, seenAt time.Time) error
	RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error
	RevokeSessionsByUserID(ctx context.Context, userID string, revokedAt time.Time) error
	RecordSuccessfulLogin(ctx context.Context, userID string, loginAt time.Time) error
	RecordFailedLogin(ctx context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error
	CreateUser(ctx context.Context, params CreateUserParams) error
	ListUsers(ctx context.Context) ([]ManagedUser, error)
	UpdateUser(ctx context.Context, params UpdateUserParams) error
	BootstrapAdmin(ctx context.Context, params BootstrapAdminParams) (bool, error)
}
