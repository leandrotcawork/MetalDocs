package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNoActor = errors.New("membership_tx: actorUserID must not be empty")

// WithMembershipContext begins a transaction, sets the required SET LOCAL GUCs for
// SECURITY DEFINER call gates, invokes fn, then commits or rolls back.
//
// GUC sequence (in order, all SET LOCAL — never SET SESSION):
//  1. SET LOCAL ROLE metaldocs_membership_writer
//  2. SET LOCAL metaldocs.actor_id = $actorUserID
//  3. SET LOCAL metaldocs.verified_capability = $capability
//
// Returns ErrNoActor if actorUserID is empty (before opening tx).
// Rollback is deferred via panic recovery; original fn error is preserved.
func WithMembershipContext(
	ctx context.Context,
	db *sql.DB,
	actorUserID string,
	capability string,
	fn func(tx *sql.Tx) error,
) (retErr error) {
	if actorUserID == "" {
		return ErrNoActor
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("membership_tx: begin: %w", err)
	}

	// Deferred rollback — fires on both panic and error return.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-panic after rollback
		}
		if retErr != nil {
			_ = tx.Rollback()
		}
	}()

	// SET LOCAL GUCs — strict order, always SET LOCAL.
	if _, err := tx.ExecContext(ctx, "SET LOCAL ROLE metaldocs_membership_writer"); err != nil {
		retErr = fmt.Errorf("membership_tx: SET LOCAL ROLE: %w", err)
		return
	}
	if _, err := tx.ExecContext(ctx, "SET LOCAL metaldocs.actor_id = $1", actorUserID); err != nil {
		retErr = fmt.Errorf("membership_tx: SET LOCAL actor_id: %w", err)
		return
	}
	if _, err := tx.ExecContext(ctx, "SET LOCAL metaldocs.verified_capability = $1", capability); err != nil {
		retErr = fmt.Errorf("membership_tx: SET LOCAL capability: %w", err)
		return
	}

	if retErr = fn(tx); retErr != nil {
		return
	}

	if err := tx.Commit(); err != nil {
		retErr = fmt.Errorf("membership_tx: commit: %w", err)
		return
	}
	return nil
}
