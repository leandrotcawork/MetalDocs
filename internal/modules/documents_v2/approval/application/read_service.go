package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"metaldocs/internal/modules/documents_v2/approval/domain"
	"metaldocs/internal/modules/documents_v2/approval/repository"
)

// ReadService exposes read-only operations for approval HTTP handlers.
type ReadService struct {
	repo repository.ApprovalRepository
}

func newReadService(repo repository.ApprovalRepository) *ReadService {
	return &ReadService{repo: repo}
}

// LoadInstance loads a single approval instance by ID for the given tenant.
func (s *ReadService) LoadInstance(ctx context.Context, db *sql.DB, tenantID, actorID, instanceID string) (*domain.Instance, error) {
	_ = actorID

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("read load instance: begin tx: %w", err)
	}
	defer tx.Rollback()

	inst, err := s.repo.LoadInstance(ctx, tx, tenantID, instanceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNoActiveInstance
		}
		return nil, err
	}
	if inst == nil {
		return nil, repository.ErrNoActiveInstance
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("read load instance: commit tx: %w", err)
	}
	return inst, nil
}

// ListPendingForActor lists inbox items pending actor action.
func (s *ReadService) ListPendingForActor(ctx context.Context, db *sql.DB, tenantID, actorID string, areaCode string, limit, offset int) ([]domain.Instance, error) {
	_ = ctx
	_ = db
	_ = tenantID
	_ = actorID
	_ = areaCode
	_ = limit
	_ = offset

	// TODO: implement inbox query - requires stage membership query.
	return []domain.Instance{}, nil
}
