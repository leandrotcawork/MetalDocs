package domain

import (
	"time"

	iamdomain "metaldocs/internal/modules/iam/domain"
)

type Identity struct {
	UserID              string
	Username            string
	Email               string
	DisplayName         string
	PasswordHash        string
	PasswordAlgo        string
	MustChangePassword  bool
	LastLoginAt         *time.Time
	FailedLoginAttempts int
	LockedUntil         *time.Time
	IsActive            bool
	Roles               []iamdomain.Role
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Session struct {
	SessionID  string
	UserID     string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	IPAddress  string
	UserAgent  string
	LastSeenAt time.Time
}

type ManagedUser struct {
	UserID              string
	Username            string
	Email               string
	DisplayName         string
	IsActive            bool
	MustChangePassword  bool
	LastLoginAt         *time.Time
	FailedLoginAttempts int
	LockedUntil         *time.Time
	Roles               []iamdomain.Role
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CreateUserParams struct {
	UserID             string
	Username           string
	Email              string
	DisplayName        string
	PasswordHash       string
	PasswordAlgo       string
	MustChangePassword bool
	IsActive           bool
	Roles              []iamdomain.Role
	CreatedBy          string
}

type UpdateUserParams struct {
	UserID             string
	DisplayName        *string
	Email              *string
	IsActive           *bool
	NewPasswordHash    *string
	MustChangePassword *bool
}

type BootstrapAdminParams struct {
	UserID             string
	Username           string
	Email              string
	DisplayName        string
	PasswordHash       string
	PasswordAlgo       string
	MustChangePassword bool
}

type CurrentUser struct {
	UserID             string           `json:"userId"`
	Username           string           `json:"username"`
	Email              string           `json:"email,omitempty"`
	DisplayName        string           `json:"displayName"`
	MustChangePassword bool             `json:"mustChangePassword"`
	Roles              []iamdomain.Role `json:"roles"`
}

type AuthenticatedSession struct {
	RawToken    string
	CurrentUser CurrentUser
	ExpiresAt   time.Time
}
