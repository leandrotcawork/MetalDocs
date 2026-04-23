package application

import (
	"context"
	"database/sql"
	"fmt"
)

// setAuthzGUC sets the tenant_id and actor_id GUC variables needed by authz.Require.
// Must be called within an open transaction, before any authz.Require calls.
func setAuthzGUC(ctx context.Context, tx *sql.Tx, tenantID, actorID string) error {
	if _, err := tx.ExecContext(ctx, "SELECT set_config('metaldocs.tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant GUC: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "SELECT set_config('metaldocs.actor_id', $1, true)", actorID); err != nil {
		return fmt.Errorf("set actor GUC: %w", err)
	}
	return nil
}
