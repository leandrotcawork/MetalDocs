package domain

import "errors"

var (
	ErrInvalidCredentials     = errors.New("auth invalid credentials")
	ErrSessionNotFound        = errors.New("auth session not found")
	ErrSessionExpired         = errors.New("auth session expired")
	ErrSessionRevoked         = errors.New("auth session revoked")
	ErrPasswordPolicy         = errors.New("auth password policy violation")
	ErrPasswordChangeRequired = errors.New("auth password change required")
	ErrIdentityLocked         = errors.New("auth identity locked")
	ErrIdentityInactive       = errors.New("auth identity inactive")
	ErrIdentityNotFound       = errors.New("auth identity not found")
	ErrUserAlreadyExists      = errors.New("auth user already exists")
)
