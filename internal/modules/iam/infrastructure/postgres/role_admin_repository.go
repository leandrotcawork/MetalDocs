package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"metaldocs/internal/modules/iam/domain"
)

type RoleAdminRepository struct {
	db *sql.DB
}

func NewRoleAdminRepository(db *sql.DB) *RoleAdminRepository {
	return &RoleAdminRepository{db: db}
}

func (r *RoleAdminRepository) HasAnyRole(ctx context.Context, role domain.Role) (bool, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM metaldocs.iam_user_roles
WHERE role_code = $1
`, string(role)).Scan(&count); err != nil {
		return false, fmt.Errorf("count role assignments: %w", err)
	}
	return count > 0, nil
}

func (r *RoleAdminRepository) UpsertUserAndAssignRole(ctx context.Context, userID, displayName string, role domain.Role, assignedBy string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin iam tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const upsertUser = `
INSERT INTO metaldocs.iam_users (user_id, display_name, is_active, updated_at)
VALUES ($1, $2, TRUE, NOW())
ON CONFLICT (user_id)
DO UPDATE SET display_name = EXCLUDED.display_name, is_active = TRUE, updated_at = NOW()
`
	if _, err := tx.ExecContext(ctx, upsertUser, userID, displayName); err != nil {
		return fmt.Errorf("upsert iam user: %w", err)
	}

	const upsertRole = `
INSERT INTO metaldocs.iam_user_roles (user_id, role_code, assigned_at, assigned_by)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (user_id, role_code)
DO UPDATE SET assigned_at = NOW(), assigned_by = EXCLUDED.assigned_by
`
	if _, err := tx.ExecContext(ctx, upsertRole, userID, string(role), assignedBy); err != nil {
		return fmt.Errorf("upsert iam role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit iam tx: %w", err)
	}
	return nil
}

func (r *RoleAdminRepository) ReplaceUserRoles(ctx context.Context, userID, displayName string, roles []domain.Role, assignedBy string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin iam replace roles tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const upsertUser = `
INSERT INTO metaldocs.iam_users (user_id, display_name, is_active, updated_at)
VALUES ($1, $2, TRUE, NOW())
ON CONFLICT (user_id)
DO UPDATE SET display_name = EXCLUDED.display_name, updated_at = NOW()
`
	if _, err := tx.ExecContext(ctx, upsertUser, userID, displayName); err != nil {
		return fmt.Errorf("upsert iam user for role replace: %w", err)
	}

	desired := make([]string, 0, len(roles))
	seen := map[string]bool{}
	for _, role := range roles {
		roleCode := strings.TrimSpace(string(role))
		if roleCode == "" || seen[roleCode] {
			continue
		}
		seen[roleCode] = true
		desired = append(desired, roleCode)
	}

	rows, err := tx.QueryContext(ctx, `SELECT role_code FROM metaldocs.iam_user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return fmt.Errorf("select existing iam roles: %w", err)
	}
	defer rows.Close()

	existing := map[string]bool{}
	for rows.Next() {
		var roleCode string
		if err := rows.Scan(&roleCode); err != nil {
			return fmt.Errorf("scan existing iam role: %w", err)
		}
		existing[roleCode] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate existing iam roles: %w", err)
	}

	desiredSet := map[string]bool{}
	for _, roleCode := range desired {
		desiredSet[roleCode] = true
		if _, err := tx.ExecContext(ctx, `
INSERT INTO metaldocs.iam_user_roles (user_id, role_code, assigned_at, assigned_by)
VALUES ($1, $2, NOW(), $3)
ON CONFLICT (user_id, role_code)
DO UPDATE SET assigned_at = NOW(), assigned_by = EXCLUDED.assigned_by
`, userID, roleCode, assignedBy); err != nil {
			return fmt.Errorf("upsert replaced iam role: %w", err)
		}
	}

	for roleCode := range existing {
		if desiredSet[roleCode] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM metaldocs.iam_user_roles WHERE user_id = $1 AND role_code = $2`, userID, roleCode); err != nil {
			return fmt.Errorf("delete stale iam role: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit iam replace roles tx: %w", err)
	}
	return nil
}
