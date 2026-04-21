package application

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"metaldocs/internal/modules/iam/domain"
)

func CheckRoleCapabilitiesVersion(ctx context.Context, db *sql.DB, tenantID string) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return fmt.Errorf("tenant id is required")
	}

	var hasTable bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.role_capabilities_versions') IS NOT NULL`).Scan(&hasTable); err != nil {
		return fmt.Errorf("check role capabilities version table: %w", err)
	}
	if !hasTable {
		return nil
	}

	var version int
	err := db.QueryRowContext(
		ctx,
		`SELECT version FROM role_capabilities_versions WHERE tenant_id = $1 LIMIT 1`,
		tenantID,
	).Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load role capabilities version: %w", err)
	}

	if version != domain.RoleCapabilitiesVersion {
		return fmt.Errorf(
			"role capabilities version mismatch: tenant=%s expected=%d got=%d",
			tenantID,
			domain.RoleCapabilitiesVersion,
			version,
		)
	}
	return nil
}
