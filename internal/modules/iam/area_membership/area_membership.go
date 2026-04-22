// Package area_membership wraps the SECURITY DEFINER DB functions
// metaldocs.grant_area_membership and metaldocs.revoke_area_membership.
// All writes to user_process_areas MUST go through this package.
package area_membership

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors returned by this package.
var (
	ErrInsufficientPrivilege = errors.New("area_membership: insufficient privilege")
	ErrMembershipNotFound    = errors.New("area_membership: membership not found")
	ErrInvalidArgument       = errors.New("area_membership: invalid argument")
)

// Membership represents an active membership row from user_process_areas.
type Membership struct {
	UserID        string
	TenantID      string
	AreaCode      string
	Role          string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	GrantedBy     string
}

// mapPgError translates *pgconn.PgError SQLSTATEs to domain sentinels.
func mapPgError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "42501": // insufficient_privilege
		return ErrInsufficientPrivilege
	case "P0002": // no_data_found
		return ErrMembershipNotFound
	case "22023": // invalid_parameter_value
		return fmt.Errorf("%w: %s", ErrInvalidArgument, pgErr.Message)
	default:
		return err
	}
}

// Grant calls metaldocs.grant_area_membership and returns the correlation UUID.
func Grant(ctx context.Context, tx *sql.Tx, tenantID, userID, areaCode, role, grantedBy string) (correlationID string, err error) {
	err = tx.QueryRowContext(ctx,
		`SELECT metaldocs.grant_area_membership($1, $2, $3, $4, $5)`,
		tenantID, userID, areaCode, role, grantedBy,
	).Scan(&correlationID)
	if err != nil {
		return "", mapPgError(err)
	}
	return correlationID, nil
}

// Revoke calls metaldocs.revoke_area_membership to soft-delete a membership.
func Revoke(ctx context.Context, tx *sql.Tx, tenantID, userID, areaCode, role, revokedBy string) error {
	_, err := tx.ExecContext(ctx,
		`SELECT metaldocs.revoke_area_membership($1, $2, $3, $4, $5)`,
		tenantID, userID, areaCode, role, revokedBy,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

// List reads active memberships for a user in a tenant (effective_to IS NULL).
func List(ctx context.Context, tx *sql.Tx, tenantID, userID string) ([]Membership, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT user_id, tenant_id, area_code, role, effective_from, effective_to, granted_by
		 FROM user_process_areas
		 WHERE tenant_id=$1 AND user_id=$2 AND effective_to IS NULL
		 ORDER BY area_code, role`,
		tenantID, userID,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var memberships []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(
			&m.UserID,
			&m.TenantID,
			&m.AreaCode,
			&m.Role,
			&m.EffectiveFrom,
			&m.EffectiveTo,
			&m.GrantedBy,
		); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	return memberships, nil
}
