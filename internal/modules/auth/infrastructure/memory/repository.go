package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	authdomain "metaldocs/internal/modules/auth/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type Repository struct {
	mu       sync.Mutex
	users    map[string]authdomain.Identity
	byLogin  map[string]string
	sessions map[string]authdomain.Session
}

func NewRepository() *Repository {
	return &Repository{
		users:    map[string]authdomain.Identity{},
		byLogin:  map[string]string{},
		sessions: map[string]authdomain.Session{},
	}
}

func (r *Repository) FindIdentityByIdentifier(_ context.Context, identifier string) (authdomain.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	userID, ok := r.byLogin[strings.ToLower(strings.TrimSpace(identifier))]
	if !ok {
		return authdomain.Identity{}, authdomain.ErrIdentityNotFound
	}
	identity, ok := r.users[userID]
	if !ok {
		return authdomain.Identity{}, authdomain.ErrIdentityNotFound
	}
	return cloneIdentity(identity), nil
}

func (r *Repository) FindIdentityByUserID(_ context.Context, userID string) (authdomain.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity, ok := r.users[strings.TrimSpace(userID)]
	if !ok {
		return authdomain.Identity{}, authdomain.ErrIdentityNotFound
	}
	return cloneIdentity(identity), nil
}

func (r *Repository) CreateSession(_ context.Context, session authdomain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.SessionID] = session
	return nil
}

func (r *Repository) FindSession(_ context.Context, sessionID string) (authdomain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return authdomain.Session{}, authdomain.ErrSessionNotFound
	}
	return session, nil
}

func (r *Repository) TouchSession(_ context.Context, sessionID string, seenAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return authdomain.ErrSessionNotFound
	}
	session.LastSeenAt = seenAt
	r.sessions[sessionID] = session
	return nil
}

func (r *Repository) RevokeSession(_ context.Context, sessionID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[sessionID]
	if !ok {
		return nil
	}
	session.RevokedAt = &revokedAt
	r.sessions[sessionID] = session
	return nil
}

func (r *Repository) RevokeSessionsByUserID(_ context.Context, userID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for sessionID, session := range r.sessions {
		if session.UserID != userID || session.RevokedAt != nil {
			continue
		}
		revokedAtUTC := revokedAt.UTC()
		session.RevokedAt = &revokedAtUTC
		r.sessions[sessionID] = session
	}
	return nil
}

func (r *Repository) RecordSuccessfulLogin(_ context.Context, userID string, loginAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity, ok := r.users[userID]
	if !ok {
		return authdomain.ErrIdentityNotFound
	}
	identity.LastLoginAt = &loginAt
	identity.FailedLoginAttempts = 0
	identity.LockedUntil = nil
	identity.UpdatedAt = loginAt
	r.users[userID] = identity
	return nil
}

func (r *Repository) RecordFailedLogin(_ context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity, ok := r.users[userID]
	if !ok {
		return authdomain.ErrIdentityNotFound
	}
	identity.FailedLoginAttempts = failedAttempts
	identity.LockedUntil = lockedUntil
	identity.UpdatedAt = time.Now().UTC()
	r.users[userID] = identity
	return nil
}

func (r *Repository) CreateUser(_ context.Context, params authdomain.CreateUserParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[params.UserID]; ok {
		return authdomain.ErrUserAlreadyExists
	}
	loginKey := strings.ToLower(params.Username)
	if loginKey == "" {
		return authdomain.ErrUserAlreadyExists
	}
	if _, ok := r.byLogin[loginKey]; ok {
		return authdomain.ErrUserAlreadyExists
	}
	if emailKey := strings.ToLower(strings.TrimSpace(params.Email)); emailKey != "" {
		if _, ok := r.byLogin[emailKey]; ok {
			return authdomain.ErrUserAlreadyExists
		}
		r.byLogin[emailKey] = params.UserID
	}
	r.byLogin[loginKey] = params.UserID
	now := time.Now().UTC()
	r.users[params.UserID] = authdomain.Identity{
		UserID:             params.UserID,
		Username:           params.Username,
		Email:              params.Email,
		DisplayName:        params.DisplayName,
		PasswordHash:       params.PasswordHash,
		PasswordAlgo:       params.PasswordAlgo,
		MustChangePassword: params.MustChangePassword,
		IsActive:           params.IsActive,
		Roles:              append([]iamdomain.Role(nil), params.Roles...),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return nil
}

func (r *Repository) ListUsers(_ context.Context) ([]authdomain.ManagedUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]authdomain.ManagedUser, 0, len(r.users))
	for _, identity := range r.users {
		out = append(out, authdomain.ManagedUser{
			UserID:              identity.UserID,
			Username:            identity.Username,
			Email:               identity.Email,
			DisplayName:         identity.DisplayName,
			IsActive:            identity.IsActive,
			MustChangePassword:  identity.MustChangePassword,
			LastLoginAt:         identity.LastLoginAt,
			FailedLoginAttempts: identity.FailedLoginAttempts,
			LockedUntil:         identity.LockedUntil,
			Roles:               append([]iamdomain.Role(nil), identity.Roles...),
			CreatedAt:           identity.CreatedAt,
			UpdatedAt:           identity.UpdatedAt,
		})
	}
	return out, nil
}

func (r *Repository) UpdateUser(_ context.Context, params authdomain.UpdateUserParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity, ok := r.users[params.UserID]
	if !ok {
		return authdomain.ErrIdentityNotFound
	}
	if params.DisplayName != nil {
		identity.DisplayName = strings.TrimSpace(*params.DisplayName)
	}
	if params.Email != nil {
		oldEmail := strings.ToLower(strings.TrimSpace(identity.Email))
		if oldEmail != "" {
			delete(r.byLogin, oldEmail)
		}
		identity.Email = strings.TrimSpace(*params.Email)
		if newEmail := strings.ToLower(strings.TrimSpace(identity.Email)); newEmail != "" {
			r.byLogin[newEmail] = identity.UserID
		}
	}
	if params.IsActive != nil {
		identity.IsActive = *params.IsActive
	}
	if params.NewPasswordHash != nil {
		identity.PasswordHash = *params.NewPasswordHash
		identity.PasswordAlgo = "bcrypt"
	}
	if params.MustChangePassword != nil {
		identity.MustChangePassword = *params.MustChangePassword
	}
	if params.ResetLockState {
		identity.FailedLoginAttempts = 0
		identity.LockedUntil = nil
	}
	identity.UpdatedAt = time.Now().UTC()
	r.users[params.UserID] = identity
	return nil
}

func (r *Repository) BootstrapAdmin(_ context.Context, params authdomain.BootstrapAdminParams) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, identity := range r.users {
		for _, role := range identity.Roles {
			if role == iamdomain.RoleAdmin {
				return false, nil
			}
		}
	}
	now := time.Now().UTC()
	r.byLogin[strings.ToLower(params.Username)] = params.UserID
	if email := strings.ToLower(strings.TrimSpace(params.Email)); email != "" {
		r.byLogin[email] = params.UserID
	}
	r.users[params.UserID] = authdomain.Identity{
		UserID:             params.UserID,
		Username:           params.Username,
		Email:              params.Email,
		DisplayName:        params.DisplayName,
		PasswordHash:       params.PasswordHash,
		PasswordAlgo:       params.PasswordAlgo,
		MustChangePassword: params.MustChangePassword,
		IsActive:           true,
		Roles:              []iamdomain.Role{iamdomain.RoleAdmin},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return true, nil
}

func (r *Repository) RolesByUserID(_ context.Context, userID string) ([]iamdomain.Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	identity, ok := r.users[userID]
	if !ok {
		return nil, iamdomain.ErrUserNotFound
	}
	if !identity.IsActive {
		return nil, iamdomain.ErrUserInactive
	}
	if len(identity.Roles) == 0 {
		return nil, iamdomain.ErrUserNotFound
	}
	return append([]iamdomain.Role(nil), identity.Roles...), nil
}

func (r *Repository) UpsertUserAndAssignRole(_ context.Context, userID, displayName string, role iamdomain.Role, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity, ok := r.users[userID]
	if !ok {
		now := time.Now().UTC()
		identity = authdomain.Identity{
			UserID:      userID,
			Username:    userID,
			DisplayName: displayName,
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		r.byLogin[strings.ToLower(userID)] = userID
	}
	if strings.TrimSpace(displayName) != "" {
		identity.DisplayName = strings.TrimSpace(displayName)
	}
	if !containsRole(identity.Roles, role) {
		identity.Roles = append(identity.Roles, role)
	}
	identity.IsActive = true
	identity.UpdatedAt = time.Now().UTC()
	r.users[userID] = identity
	return nil
}

func (r *Repository) ReplaceUserRoles(_ context.Context, userID, displayName string, roles []iamdomain.Role, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	identity, ok := r.users[userID]
	if !ok {
		now := time.Now().UTC()
		identity = authdomain.Identity{
			UserID:      userID,
			Username:    userID,
			DisplayName: displayName,
			IsActive:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		r.byLogin[strings.ToLower(userID)] = userID
	}
	if strings.TrimSpace(displayName) != "" {
		identity.DisplayName = strings.TrimSpace(displayName)
	}
	identity.Roles = append([]iamdomain.Role(nil), roles...)
	identity.IsActive = true
	identity.UpdatedAt = time.Now().UTC()
	r.users[userID] = identity
	return nil
}

func cloneIdentity(identity authdomain.Identity) authdomain.Identity {
	identity.Roles = append([]iamdomain.Role(nil), identity.Roles...)
	return identity
}

func containsRole(roles []iamdomain.Role, want iamdomain.Role) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}
