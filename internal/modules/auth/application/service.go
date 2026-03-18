package application

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	authdomain "metaldocs/internal/modules/auth/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"

	"golang.org/x/crypto/bcrypt"
)

const passwordAlgoBcrypt = "bcrypt"

type Config struct {
	SessionCookieName      string
	SessionTTL             time.Duration
	SessionSecret          string
	PasswordMinLength      int
	LoginMaxFailedAttempts int
	LoginLockDuration      time.Duration
	LegacyHeaderEnabled    bool
	OriginProtection       bool
	TrustedOrigins         []string
	BootstrapAdminEnabled  bool
	BootstrapAdminUserID   string
	BootstrapAdminUsername string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	BootstrapAdminName     string
	CookieSecure           bool
}

type Service struct {
	repo         authdomain.Repository
	roleProvider iamdomain.RoleProvider
	cfg          Config
}

func NewService(repo authdomain.Repository, roleProvider iamdomain.RoleProvider, cfg Config) *Service {
	return &Service{repo: repo, roleProvider: roleProvider, cfg: cfg}
}

func (s *Service) BootstrapLocalAdmin(ctx context.Context) error {
	if !s.cfg.BootstrapAdminEnabled || strings.TrimSpace(s.cfg.BootstrapAdminPassword) == "" {
		return nil
	}

	passwordHash, err := s.hashPassword(strings.TrimSpace(s.cfg.BootstrapAdminPassword))
	if err != nil {
		return err
	}

	_, err = s.repo.BootstrapAdmin(ctx, authdomain.BootstrapAdminParams{
		UserID:             strings.TrimSpace(s.cfg.BootstrapAdminUserID),
		Username:           strings.TrimSpace(s.cfg.BootstrapAdminUsername),
		Email:              strings.TrimSpace(s.cfg.BootstrapAdminEmail),
		DisplayName:        strings.TrimSpace(s.cfg.BootstrapAdminName),
		PasswordHash:       passwordHash,
		PasswordAlgo:       passwordAlgoBcrypt,
		MustChangePassword: true,
	})
	return err
}

func (s *Service) Authenticate(ctx context.Context, identifier, password string, r *http.Request) (authdomain.AuthenticatedSession, error) {
	identifier = strings.TrimSpace(identifier)
	password = strings.TrimSpace(password)
	if identifier == "" || password == "" {
		return authdomain.AuthenticatedSession{}, authdomain.ErrInvalidCredentials
	}

	identity, err := s.repo.FindIdentityByIdentifier(ctx, identifier)
	if err != nil {
		return authdomain.AuthenticatedSession{}, err
	}
	if !identity.IsActive {
		return authdomain.AuthenticatedSession{}, authdomain.ErrIdentityInactive
	}
	if identity.LockedUntil != nil && identity.LockedUntil.After(time.Now().UTC()) {
		return authdomain.AuthenticatedSession{}, authdomain.ErrIdentityLocked
	}
	if bcrypt.CompareHashAndPassword([]byte(identity.PasswordHash), []byte(password)) != nil {
		attempts := identity.FailedLoginAttempts + 1
		var lockedUntil *time.Time
		if attempts >= s.cfg.LoginMaxFailedAttempts {
			lock := time.Now().UTC().Add(s.cfg.LoginLockDuration)
			lockedUntil = &lock
		}
		_ = s.repo.RecordFailedLogin(ctx, identity.UserID, attempts, lockedUntil)
		return authdomain.AuthenticatedSession{}, authdomain.ErrInvalidCredentials
	}

	now := time.Now().UTC()
	if err := s.repo.RecordSuccessfulLogin(ctx, identity.UserID, now); err != nil {
		return authdomain.AuthenticatedSession{}, err
	}

	rawToken, sessionID, err := s.newSessionToken()
	if err != nil {
		return authdomain.AuthenticatedSession{}, err
	}
	session := authdomain.Session{
		SessionID:  sessionID,
		UserID:     identity.UserID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.cfg.SessionTTL),
		IPAddress:  remoteIP(r),
		UserAgent:  truncate(strings.TrimSpace(r.UserAgent()), 512),
		LastSeenAt: now,
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return authdomain.AuthenticatedSession{}, err
	}

	user, err := s.buildCurrentUser(ctx, identity.UserID)
	if err != nil {
		return authdomain.AuthenticatedSession{}, err
	}

	return authdomain.AuthenticatedSession{
		RawToken:    rawToken,
		CurrentUser: user,
		ExpiresAt:   session.ExpiresAt,
	}, nil
}

func (s *Service) ResolveSession(ctx context.Context, rawToken string) (authdomain.CurrentUser, error) {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return authdomain.CurrentUser{}, authdomain.ErrSessionNotFound
	}

	sessionID, err := s.tokenHashFromCookieValue(token)
	if err != nil {
		return authdomain.CurrentUser{}, authdomain.ErrSessionNotFound
	}

	session, err := s.repo.FindSession(ctx, sessionID)
	if err != nil {
		return authdomain.CurrentUser{}, err
	}
	if session.RevokedAt != nil {
		return authdomain.CurrentUser{}, authdomain.ErrSessionRevoked
	}
	if session.ExpiresAt.Before(time.Now().UTC()) {
		return authdomain.CurrentUser{}, authdomain.ErrSessionExpired
	}
	if err := s.repo.TouchSession(ctx, sessionID, time.Now().UTC()); err != nil {
		return authdomain.CurrentUser{}, err
	}
	return s.buildCurrentUser(ctx, session.UserID)
}

func (s *Service) Logout(ctx context.Context, rawToken string) error {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return nil
	}
	sessionID, err := s.tokenHashFromCookieValue(token)
	if err != nil {
		return nil
	}
	return s.repo.RevokeSession(ctx, sessionID, time.Now().UTC())
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return s.ChangePasswordForUser(ctx, authdomain.CurrentUser{UserID: userID}, currentPassword, newPassword)
}

func (s *Service) ChangePasswordForUser(ctx context.Context, currentUser authdomain.CurrentUser, currentPassword, newPassword string) error {
	userID := strings.TrimSpace(currentUser.UserID)
	userID = strings.TrimSpace(userID)
	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	if userID == "" {
		return authdomain.ErrInvalidCredentials
	}
	if err := s.validatePassword(newPassword); err != nil {
		return err
	}

	identity, err := s.repo.FindIdentityByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if !currentUser.MustChangePassword {
		if currentPassword == "" {
			return authdomain.ErrInvalidCredentials
		}
		if bcrypt.CompareHashAndPassword([]byte(identity.PasswordHash), []byte(currentPassword)) != nil {
			return authdomain.ErrInvalidCredentials
		}
	}
	if currentUser.MustChangePassword && currentPassword != "" && bcrypt.CompareHashAndPassword([]byte(identity.PasswordHash), []byte(currentPassword)) != nil {
		return authdomain.ErrInvalidCredentials
	}

	passwordHash, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}

	required := false
	if err := s.repo.UpdateUser(ctx, authdomain.UpdateUserParams{
		UserID:             userID,
		NewPasswordHash:    &passwordHash,
		MustChangePassword: &required,
	}); err != nil {
		return err
	}
	return nil
}

func (s *Service) ListUsers(ctx context.Context) ([]authdomain.ManagedUser, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) CreateUser(ctx context.Context, userID, username, email, displayName, password string, roles []iamdomain.Role, createdBy string) error {
	userID = strings.TrimSpace(userID)
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)
	createdBy = strings.TrimSpace(createdBy)
	if userID == "" {
		userID = username
	}
	if username == "" {
		return authdomain.ErrUserAlreadyExists
	}
	if displayName == "" {
		displayName = username
	}
	if createdBy == "" {
		createdBy = "system"
	}
	if err := s.validatePassword(password); err != nil {
		return err
	}
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return err
	}

	return s.repo.CreateUser(ctx, authdomain.CreateUserParams{
		UserID:             userID,
		Username:           username,
		Email:              email,
		DisplayName:        displayName,
		PasswordHash:       passwordHash,
		PasswordAlgo:       passwordAlgoBcrypt,
		MustChangePassword: true,
		IsActive:           true,
		Roles:              roles,
		CreatedBy:          createdBy,
	})
}

func (s *Service) UpdateUser(ctx context.Context, params authdomain.UpdateUserParams, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if newPassword != "" {
		if err := s.validatePassword(newPassword); err != nil {
			return err
		}
		passwordHash, err := s.hashPassword(newPassword)
		if err != nil {
			return err
		}
		params.NewPasswordHash = &passwordHash
	}
	return s.repo.UpdateUser(ctx, params)
}

func (s *Service) AdminResetPassword(ctx context.Context, userID, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if err := s.validatePassword(newPassword); err != nil {
		return err
	}
	passwordHash, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}
	required := true
	if err := s.repo.UpdateUser(ctx, authdomain.UpdateUserParams{
		UserID:             strings.TrimSpace(userID),
		NewPasswordHash:    &passwordHash,
		MustChangePassword: &required,
		ResetLockState:     true,
	}); err != nil {
		return err
	}
	return s.repo.RevokeSessionsByUserID(ctx, strings.TrimSpace(userID), time.Now().UTC())
}

func (s *Service) UnlockUser(ctx context.Context, userID string) error {
	return s.repo.UpdateUser(ctx, authdomain.UpdateUserParams{
		UserID:         strings.TrimSpace(userID),
		ResetLockState: true,
	})
}

func (s *Service) SessionCookie(rawToken string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure,
		Expires:  expiresAt.UTC(),
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	}
}

func (s *Service) SessionCookieName() string {
	return s.cfg.SessionCookieName
}

func (s *Service) ExpiredSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	}
}

func (s *Service) CurrentUser(ctx context.Context, userID string) (authdomain.CurrentUser, error) {
	return s.buildCurrentUser(ctx, userID)
}

func (s *Service) buildCurrentUser(ctx context.Context, userID string) (authdomain.CurrentUser, error) {
	identity, err := s.repo.FindIdentityByUserID(ctx, userID)
	if err != nil {
		return authdomain.CurrentUser{}, err
	}
	roles, err := s.roleProvider.RolesByUserID(ctx, userID)
	if err != nil {
		return authdomain.CurrentUser{}, err
	}
	return authdomain.CurrentUser{
		UserID:             identity.UserID,
		Username:           identity.Username,
		Email:              identity.Email,
		DisplayName:        identity.DisplayName,
		MustChangePassword: identity.MustChangePassword,
		Roles:              roles,
	}, nil
}

func (s *Service) validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < s.cfg.PasswordMinLength {
		return fmt.Errorf("%w: password must contain at least %d characters", authdomain.ErrPasswordPolicy, s.cfg.PasswordMinLength)
	}
	return nil
}

func (s *Service) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (s *Service) newSessionToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	sig := s.signToken(token)
	cookieValue := token + "." + sig
	return cookieValue, hashToken(token), nil
}

func (s *Service) tokenHashFromCookieValue(raw string) (string, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", authdomain.ErrSessionNotFound
	}
	if !hmac.Equal([]byte(parts[1]), []byte(s.signToken(parts[0]))) {
		return "", authdomain.ErrSessionNotFound
	}
	return hashToken(parts[0]), nil
}

func (s *Service) signToken(token string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.SessionSecret))
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func remoteIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return truncate(strings.TrimSpace(parts[0]), 128)
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return truncate(host, 128)
	}
	return truncate(strings.TrimSpace(r.RemoteAddr), 128)
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
