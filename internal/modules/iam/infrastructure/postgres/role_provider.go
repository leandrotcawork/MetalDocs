package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"metaldocs/internal/modules/iam/domain"
)

type RoleProvider struct {
	db *sql.DB
}

func NewRoleProvider(db *sql.DB) *RoleProvider {
	return &RoleProvider{db: db}
}

func (p *RoleProvider) RolesByUserID(ctx context.Context, userID string) ([]domain.Role, error) {
	const checkUserSQL = `
SELECT is_active
FROM metaldocs.iam_users
WHERE user_id = $1
`
	var isActive bool
	if err := p.db.QueryRowContext(ctx, checkUserSQL, userID).Scan(&isActive); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("check iam user: %w", err)
	}
	if !isActive {
		return nil, domain.ErrUserInactive
	}

	const rolesSQL = `
SELECT role_code
FROM metaldocs.iam_user_roles
WHERE user_id = $1
ORDER BY role_code ASC
`
	rows, err := p.db.QueryContext(ctx, rolesSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("query iam roles: %w", err)
	}
	defer rows.Close()

	roles := make([]domain.Role, 0, 4)
	for rows.Next() {
		var roleCode string
		if err := rows.Scan(&roleCode); err != nil {
			return nil, fmt.Errorf("scan iam role: %w", err)
		}
		roles = append(roles, domain.Role(roleCode))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate iam roles: %w", err)
	}
	if len(roles) == 0 {
		return nil, domain.ErrUserNotFound
	}

	return roles, nil
}
