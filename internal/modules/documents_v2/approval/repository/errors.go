package repository

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrStaleRevision         = errors.New("approval: stale revision — concurrent modification detected")
	ErrNoActiveInstance      = errors.New("approval: no active approval instance for document")
	ErrDuplicateSubmission   = errors.New("approval: duplicate submission idempotency key")
	ErrActorAlreadySigned    = errors.New("approval: actor already signed this instance")
	ErrCrossTenantSignoff    = errors.New("approval: cross-tenant signoff rejected")
	ErrInstanceCompleted     = errors.New("approval: instance is already terminal")
	ErrStageNotActive        = errors.New("approval: stage is not in expected status")
	ErrFKViolation           = errors.New("approval: foreign key violation")
	ErrCheckViolation        = errors.New("approval: check constraint violation")
	ErrInsufficientPrivilege = errors.New("approval: insufficient privilege — GUC context missing")
	ErrUnknownDB             = errors.New("approval: unknown database error")
	ErrRouteInUse            = errors.New("approval: route is referenced by one or more instances and cannot be modified")
	ErrDuplicateRouteProfile = errors.New("approval: a route already exists for this tenant+profile combination")
)

// MapHints carries constraint-name hints for SQLSTATE 23505 disambiguation.
type MapHints struct {
	UniqueConstraint string // e.g. "ux_approval_instances_active"
}

// MapPgError translates *pgconn.PgError to domain errors.
// Falls back to ErrUnknownDB (wrapping original) for unrecognized codes.
func MapPgError(err error, hints MapHints) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505": // unique_violation
		switch pgErr.ConstraintName {
		case "ux_approval_instances_active", "approval_instances_document_v2_id_idempotency_key_key":
			return ErrDuplicateSubmission
		case "approval_signoffs_approval_instance_id_actor_user_id_key",
			"approval_signoffs_stage_instance_id_actor_user_id_key":
			return ErrActorAlreadySigned
		case "approval_routes_tenant_profile_key":
			return ErrDuplicateRouteProfile
		default:
			if hints.UniqueConstraint != "" && pgErr.ConstraintName == hints.UniqueConstraint {
				return ErrDuplicateSubmission
			}
			return ErrActorAlreadySigned
		}
	case "23503": // foreign_key_violation
		return ErrFKViolation
	case "23514": // check_violation
		if pgErr.Message != "" {
			return fmt.Errorf("%w: %s", ErrCheckViolation, pgErr.Message)
		}
		return ErrCheckViolation
	case "42501": // insufficient_privilege
		return ErrInsufficientPrivilege
	case "P0001":
		// enforce_route_immutable() sets message prefix "ErrRouteInUse:" as a stable
		// contract token (migration 0145). String check is intentional; if the trigger
		// message text ever changes, update both here and the migration.
		if strings.HasPrefix(pgErr.Message, "ErrRouteInUse") {
			return ErrRouteInUse
		}
		return fmt.Errorf("%w: %s (SQLSTATE %s)", ErrUnknownDB, pgErr.Message, pgErr.Code)
	default:
		return fmt.Errorf("%w: %s (SQLSTATE %s)", ErrUnknownDB, pgErr.Message, pgErr.Code)
	}
}
