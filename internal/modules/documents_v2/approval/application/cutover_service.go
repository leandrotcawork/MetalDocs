package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrLegacyDocumentsRemain is returned by ValidateLegacyCutoverReady when one
// or more documents still carry a legacy status ('finalized' or 'archived').
// Those documents must be migrated to a Spec-2 status before the compat window
// can be safely removed by migration 0142.
var ErrLegacyDocumentsRemain = errors.New("cutover: legacy documents remain with status 'finalized' or 'archived'")

// CutoverService validates preconditions for the Phase 5.10 legacy cutover.
//
// The compat window itself (the two extra OR clauses in the transition trigger
// installed by migration 0133) is removed by the DDL migration 0142. This
// service only enforces the safety check that must pass before that migration
// is applied: no documents may remain in a legacy status.
type CutoverService struct {
	emitter EventEmitter
	clock   Clock
}

// NewCutoverService constructs a CutoverService.
func NewCutoverService(emitter EventEmitter, clock Clock) *CutoverService {
	return &CutoverService{emitter: emitter, clock: clock}
}

// ValidateLegacyCutoverReady checks that no documents carry the legacy
// 'finalized' or 'archived' status. If any remain, it returns an error that
// wraps ErrLegacyDocumentsRemain and includes the count so the caller can
// surface a meaningful message.
//
// This method is read-only; it does not modify any state. Run it (and confirm
// it returns nil) before applying migration 0142_disable_legacy_compat.sql.
func (s *CutoverService) ValidateLegacyCutoverReady(ctx context.Context, db *sql.DB) error {
	var count int64
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM documents WHERE status IN ('finalized','archived')`,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("cutover: count legacy documents: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("%w (count: %d)", ErrLegacyDocumentsRemain, count)
	}
	return nil
}
