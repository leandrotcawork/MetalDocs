package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	authdomain "metaldocs/internal/modules/auth/domain"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindIdentityByIdentifier(ctx context.Context, identifier string) (authdomain.Identity, error) {
	const q = `
SELECT i.user_id, i.username, COALESCE(i.email, ''), u.display_name, i.password_hash, i.password_algo,
       i.must_change_password, i.last_login_at, i.failed_login_attempts, i.locked_until, u.is_active,
       i.created_at, i.updated_at
FROM metaldocs.auth_identities i
JOIN metaldocs.iam_users u ON u.user_id = i.user_id
WHERE lower(i.username) = lower($1) OR lower(COALESCE(i.email, '')) = lower($1)
`
	return r.loadIdentity(ctx, q, identifier)
}

func (r *Repository) FindIdentityByUserID(ctx context.Context, userID string) (authdomain.Identity, error) {
	const q = `
SELECT i.user_id, i.username, COALESCE(i.email, ''), u.display_name, i.password_hash, i.password_algo,
       i.must_change_password, i.last_login_at, i.failed_login_attempts, i.locked_until, u.is_active,
       i.created_at, i.updated_at
FROM metaldocs.auth_identities i
JOIN metaldocs.iam_users u ON u.user_id = i.user_id
WHERE i.user_id = $1
`
	return r.loadIdentity(ctx, q, userID)
}

func (r *Repository) CreateSession(ctx context.Context, session authdomain.Session) error {
	const q = `
INSERT INTO metaldocs.auth_sessions (session_id, user_id, created_at, expires_at, revoked_at, ip_address, user_agent, last_seen_at)
VALUES ($1, $2, $3, $4, NULL, $5, $6, $7)
`
	_, err := r.db.ExecContext(ctx, q, session.SessionID, session.UserID, session.CreatedAt, session.ExpiresAt, session.IPAddress, session.UserAgent, session.LastSeenAt)
	if err != nil {
		return fmt.Errorf("insert auth session: %w", err)
	}
	return nil
}

func (r *Repository) FindSession(ctx context.Context, sessionID string) (authdomain.Session, error) {
	const q = `
SELECT session_id, user_id, created_at, expires_at, revoked_at, COALESCE(ip_address, ''), COALESCE(user_agent, ''), last_seen_at
FROM metaldocs.auth_sessions
WHERE session_id = $1
`
	var session authdomain.Session
	if err := r.db.QueryRowContext(ctx, q, sessionID).Scan(
		&session.SessionID,
		&session.UserID,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.RevokedAt,
		&session.IPAddress,
		&session.UserAgent,
		&session.LastSeenAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return authdomain.Session{}, authdomain.ErrSessionNotFound
		}
		return authdomain.Session{}, fmt.Errorf("select auth session: %w", err)
	}
	return session, nil
}

func (r *Repository) TouchSession(ctx context.Context, sessionID string, seenAt time.Time) error {
	const q = `
UPDATE metaldocs.auth_sessions
SET last_seen_at = $2
WHERE session_id = $1
`
	_, err := r.db.ExecContext(ctx, q, sessionID, seenAt)
	if err != nil {
		return fmt.Errorf("touch auth session: %w", err)
	}
	return nil
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID string, revokedAt time.Time) error {
	const q = `
UPDATE metaldocs.auth_sessions
SET revoked_at = $2
WHERE session_id = $1
`
	_, err := r.db.ExecContext(ctx, q, sessionID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}
	return nil
}

func (r *Repository) RecordSuccessfulLogin(ctx context.Context, userID string, loginAt time.Time) error {
	const q = `
UPDATE metaldocs.auth_identities
SET last_login_at = $2,
    failed_login_attempts = 0,
    locked_until = NULL,
    updated_at = $2
WHERE user_id = $1
`
	_, err := r.db.ExecContext(ctx, q, userID, loginAt)
	if err != nil {
		return fmt.Errorf("update successful login: %w", err)
	}
	return nil
}

func (r *Repository) RecordFailedLogin(ctx context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error {
	const q = `
UPDATE metaldocs.auth_identities
SET failed_login_attempts = $2,
    locked_until = $3,
    updated_at = NOW()
WHERE user_id = $1
`
	_, err := r.db.ExecContext(ctx, q, userID, failedAttempts, lockedUntil)
	if err != nil {
		return fmt.Errorf("update failed login: %w", err)
	}
	return nil
}

func (r *Repository) CreateUser(ctx context.Context, params authdomain.CreateUserParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create user tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensureUniqueIdentity(ctx, tx, params.UserID, params.Username, params.Email); err != nil {
		return err
	}

	const insertUser = `
INSERT INTO metaldocs.iam_users (user_id, display_name, is_active, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
`
	if _, err := tx.ExecContext(ctx, insertUser, params.UserID, params.DisplayName, params.IsActive); err != nil {
		return fmt.Errorf("insert iam user: %w", err)
	}

	const insertIdentity = `
INSERT INTO metaldocs.auth_identities (user_id, username, email, password_hash, password_algo, must_change_password, last_login_at, failed_login_attempts, locked_until, created_at, updated_at)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, NULL, 0, NULL, NOW(), NOW())
`
	if _, err := tx.ExecContext(ctx, insertIdentity, params.UserID, params.Username, params.Email, params.PasswordHash, params.PasswordAlgo, params.MustChangePassword); err != nil {
		return fmt.Errorf("insert auth identity: %w", err)
	}

	for _, role := range uniqueRoles(params.Roles) {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.iam_user_roles (user_id, role_code, assigned_at, assigned_by)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (user_id, role_code)
DO UPDATE SET assigned_at = NOW(), assigned_by = EXCLUDED.assigned_by
`, params.UserID, string(role), params.CreatedBy); err != nil {
			return fmt.Errorf("insert iam user role: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create user tx: %w", err)
	}
	return nil
}

func (r *Repository) ListUsers(ctx context.Context) ([]authdomain.ManagedUser, error) {
	const q = `
SELECT u.user_id, i.username, COALESCE(i.email, ''), u.display_name, u.is_active, i.must_change_password,
       i.last_login_at, i.failed_login_attempts, i.locked_until, u.created_at, u.updated_at,
       COALESCE(string_agg(r.role_code, ',' ORDER BY r.role_code) FILTER (WHERE r.role_code IS NOT NULL), '')
FROM metaldocs.iam_users u
JOIN metaldocs.auth_identities i ON i.user_id = u.user_id
LEFT JOIN metaldocs.iam_user_roles r ON r.user_id = u.user_id
GROUP BY u.user_id, i.username, i.email, u.display_name, u.is_active, i.must_change_password, i.last_login_at, i.failed_login_attempts, i.locked_until, u.created_at, u.updated_at
ORDER BY u.created_at DESC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list managed users: %w", err)
	}
	defer rows.Close()

	items := make([]authdomain.ManagedUser, 0)
	for rows.Next() {
		var item authdomain.ManagedUser
		var rolesCSV string
		if err := rows.Scan(
			&item.UserID,
			&item.Username,
			&item.Email,
			&item.DisplayName,
			&item.IsActive,
			&item.MustChangePassword,
			&item.LastLoginAt,
			&item.FailedLoginAttempts,
			&item.LockedUntil,
			&item.CreatedAt,
			&item.UpdatedAt,
			&rolesCSV,
		); err != nil {
			return nil, fmt.Errorf("scan managed user: %w", err)
		}
		item.Roles = csvRolesToDomain(rolesCSV)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate managed users: %w", err)
	}
	return items, nil
}

func (r *Repository) UpdateUser(ctx context.Context, params authdomain.UpdateUserParams) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update user tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if params.DisplayName != nil || params.IsActive != nil {
		if _, err := tx.ExecContext(ctx, `
UPDATE metaldocs.iam_users
SET display_name = COALESCE($2, display_name),
    is_active = COALESCE($3, is_active),
    updated_at = NOW()
WHERE user_id = $1
`, params.UserID, nullableText(params.DisplayName), nullableBool(params.IsActive)); err != nil {
			return fmt.Errorf("update iam user: %w", err)
		}
	}

	if params.Email != nil || params.NewPasswordHash != nil || params.MustChangePassword != nil {
		if params.Email != nil {
			if err := ensureUniqueIdentity(ctx, tx, params.UserID, "", strings.TrimSpace(*params.Email)); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE metaldocs.auth_identities
SET email = COALESCE($2, email),
    password_hash = COALESCE($3, password_hash),
    password_algo = CASE WHEN $3 IS NULL THEN password_algo ELSE 'bcrypt' END,
    must_change_password = COALESCE($4, must_change_password),
    failed_login_attempts = CASE WHEN $3 IS NULL THEN failed_login_attempts ELSE 0 END,
    locked_until = CASE WHEN $3 IS NULL THEN locked_until ELSE NULL END,
    updated_at = NOW()
WHERE user_id = $1
`, params.UserID, nullableText(params.Email), nullableText(params.NewPasswordHash), nullableBool(params.MustChangePassword)); err != nil {
			return fmt.Errorf("update auth identity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update user tx: %w", err)
	}
	return nil
}

func (r *Repository) BootstrapAdmin(ctx context.Context, params authdomain.BootstrapAdminParams) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin bootstrap admin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var adminCount int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM metaldocs.iam_user_roles
WHERE role_code = 'admin'
`).Scan(&adminCount); err != nil {
		return false, fmt.Errorf("count admin roles: %w", err)
	}
	if adminCount > 0 {
		return false, nil
	}

	if err := ensureUniqueIdentity(ctx, tx, params.UserID, params.Username, params.Email); err != nil {
		return false, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.iam_users (user_id, display_name, is_active, created_at, updated_at)
VALUES ($1, $2, TRUE, NOW(), NOW())
ON CONFLICT (user_id)
DO UPDATE SET display_name = EXCLUDED.display_name, is_active = TRUE, updated_at = NOW()
`, params.UserID, params.DisplayName); err != nil {
		return false, fmt.Errorf("bootstrap iam user: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.auth_identities (user_id, username, email, password_hash, password_algo, must_change_password, last_login_at, failed_login_attempts, locked_until, created_at, updated_at)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, NULL, 0, NULL, NOW(), NOW())
ON CONFLICT (user_id)
DO UPDATE SET username = EXCLUDED.username,
              email = EXCLUDED.email,
              password_hash = EXCLUDED.password_hash,
              password_algo = EXCLUDED.password_algo,
              must_change_password = EXCLUDED.must_change_password,
              updated_at = NOW()
`, params.UserID, params.Username, params.Email, params.PasswordHash, params.PasswordAlgo, params.MustChangePassword); err != nil {
		return false, fmt.Errorf("bootstrap auth identity: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.iam_user_roles (user_id, role_code, assigned_at, assigned_by)
VALUES ($1, 'admin', NOW(), 'bootstrap')
ON CONFLICT (user_id, role_code)
DO UPDATE SET assigned_at = NOW(), assigned_by = EXCLUDED.assigned_by
`, params.UserID); err != nil {
		return false, fmt.Errorf("bootstrap admin role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit bootstrap admin tx: %w", err)
	}
	return true, nil
}

func (r *Repository) loadIdentity(ctx context.Context, query string, arg string) (authdomain.Identity, error) {
	var identity authdomain.Identity
	if err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&identity.UserID,
		&identity.Username,
		&identity.Email,
		&identity.DisplayName,
		&identity.PasswordHash,
		&identity.PasswordAlgo,
		&identity.MustChangePassword,
		&identity.LastLoginAt,
		&identity.FailedLoginAttempts,
		&identity.LockedUntil,
		&identity.IsActive,
		&identity.CreatedAt,
		&identity.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return authdomain.Identity{}, authdomain.ErrIdentityNotFound
		}
		return authdomain.Identity{}, fmt.Errorf("load auth identity: %w", err)
	}

	roles, err := r.loadRoles(ctx, identity.UserID)
	if err != nil {
		return authdomain.Identity{}, err
	}
	identity.Roles = roles
	return identity, nil
}

func (r *Repository) loadRoles(ctx context.Context, userID string) ([]iamdomain.Role, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT role_code
FROM metaldocs.iam_user_roles
WHERE user_id = $1
ORDER BY role_code ASC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("load auth roles: %w", err)
	}
	defer rows.Close()

	roles := make([]iamdomain.Role, 0, 4)
	for rows.Next() {
		var roleCode string
		if err := rows.Scan(&roleCode); err != nil {
			return nil, fmt.Errorf("scan auth role: %w", err)
		}
		roles = append(roles, iamdomain.Role(roleCode))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auth roles: %w", err)
	}
	return roles, nil
}

func ensureUniqueIdentity(ctx context.Context, tx *sql.Tx, userID, username, email string) error {
	if strings.TrimSpace(username) != "" {
		var otherUserID string
		err := tx.QueryRowContext(ctx, `
SELECT user_id
FROM metaldocs.auth_identities
WHERE lower(username) = lower($1)
`, username).Scan(&otherUserID)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("check username uniqueness: %w", err)
		}
		if err == nil && otherUserID != strings.TrimSpace(userID) {
			return authdomain.ErrUserAlreadyExists
		}
	}

	if strings.TrimSpace(email) != "" {
		var otherUserID string
		err := tx.QueryRowContext(ctx, `
SELECT user_id
FROM metaldocs.auth_identities
WHERE lower(email) = lower($1)
`, email).Scan(&otherUserID)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("check email uniqueness: %w", err)
		}
		if err == nil && otherUserID != strings.TrimSpace(userID) {
			return authdomain.ErrUserAlreadyExists
		}
	}
	return nil
}

func uniqueRoles(in []iamdomain.Role) []iamdomain.Role {
	seen := make(map[iamdomain.Role]struct{}, len(in))
	out := make([]iamdomain.Role, 0, len(in))
	for _, role := range in {
		role = iamdomain.Role(strings.TrimSpace(string(role)))
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func stringRolesToDomain(roles []string) []iamdomain.Role {
	out := make([]iamdomain.Role, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		out = append(out, iamdomain.Role(role))
	}
	return out
}

func csvRolesToDomain(rolesCSV string) []iamdomain.Role {
	if strings.TrimSpace(rolesCSV) == "" {
		return nil
	}
	return stringRolesToDomain(strings.Split(rolesCSV, ","))
}

func nullableText(value *string) any {
	if value == nil {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableBool(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}
