package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"metaldocs/internal/modules/iam/domain"
)

type RoleAdminRepository struct {
	db *sql.DB
}

func NewRoleAdminRepository(db *sql.DB) *RoleAdminRepository {
	return &RoleAdminRepository{db: db}
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
