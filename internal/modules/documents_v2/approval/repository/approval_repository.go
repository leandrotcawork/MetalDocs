package repository

import (
	"context"
	"database/sql"
	"time"

	"metaldocs/internal/modules/documents_v2/approval/domain"
)

// SignoffInsertResult returned by InsertSignoff.
type SignoffInsertResult struct {
	ID        string
	WasReplay bool // true if ON CONFLICT detected existing matching signoff
}

// ScheduledPublishRow returned by ListScheduledDue.
type ScheduledPublishRow struct {
	DocumentID      string
	TenantID        string
	EffectiveFrom   time.Time
	RevisionVersion int
}

// ApprovalRepository defines all persistence operations for the approval subsystem.
// All mutating methods take *sql.Tx — callers own tx lifecycle (Phase 5 services).
// ListScheduledDue requires tx opened with sql.LevelReadCommitted (F6).
type ApprovalRepository interface {
	InsertInstance(ctx context.Context, tx *sql.Tx, inst domain.Instance) error
	InsertStageInstances(ctx context.Context, tx *sql.Tx, stages []domain.StageInstance) error
	InsertSignoff(ctx context.Context, tx *sql.Tx, s domain.Signoff) (SignoffInsertResult, error)
	LoadSignoffByActor(ctx context.Context, tx *sql.Tx, tenantID, instanceID, actorUserID string) (*domain.Signoff, error)
	LoadInstance(ctx context.Context, tx *sql.Tx, tenantID, id string) (*domain.Instance, error)
	LoadActiveInstanceByDocument(ctx context.Context, tx *sql.Tx, tenantID, docID string) (*domain.Instance, error)
	UpdateStageStatus(ctx context.Context, tx *sql.Tx, tenantID, stageID string, newStatus, expectedOldStatus domain.StageStatus) error
	UpdateInstanceStatus(ctx context.Context, tx *sql.Tx, tenantID, instID string, newStatus domain.InstanceStatus, expectedStatus domain.InstanceStatus, completedAt *time.Time) error
	// ListScheduledDue caller MUST use READ COMMITTED isolation; SKIP LOCKED semantics require it.
	ListScheduledDue(ctx context.Context, tx *sql.Tx, now time.Time, limit int) ([]ScheduledPublishRow, error)
}
