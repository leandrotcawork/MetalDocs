package application

import (
	"context"
	"database/sql"
	"encoding/json"
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

// LoadActiveInstanceByDocument finds the current active approval instance for a document.
func (s *ReadService) LoadActiveInstanceByDocument(ctx context.Context, db *sql.DB, tenantID, documentID string) (*domain.Instance, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("read load instance by document: begin tx: %w", err)
	}
	defer tx.Rollback()

	inst, err := s.repo.LoadActiveInstanceByDocument(ctx, tx, tenantID, documentID)
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
		return nil, fmt.Errorf("read load instance by document: commit tx: %w", err)
	}
	return inst, nil
}

// ListPendingForActor lists inbox items pending actor action.
func (s *ReadService) ListPendingForActor(ctx context.Context, db *sql.DB, tenantID, actorID string, areaCode string, limit, offset int) ([]domain.Instance, error) {
	if limit <= 0 {
		limit = 25
	}

	actorJSON, err := json.Marshal([]string{actorID})
	if err != nil {
		return nil, fmt.Errorf("list pending: marshal actor: %w", err)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("list pending: begin tx: %w", err)
	}
	defer tx.Rollback()

	const q = `
		SELECT DISTINCT ai.id
		FROM approval_instances ai
		JOIN approval_stage_instances asi ON asi.approval_instance_id = ai.id
		WHERE ai.tenant_id = $1
		  AND ai.status = 'in_progress'
		  AND asi.status = 'active'
		  AND asi.eligible_actor_ids @> $2::jsonb
		  AND ($3 = '' OR asi.area_code_snapshot = $3)
		ORDER BY ai.id
		LIMIT $4 OFFSET $5`

	rows, err := tx.QueryContext(ctx, q, tenantID, string(actorJSON), areaCode, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list pending: query: %w", err)
	}

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("list pending: scan id: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list pending: rows: %w", err)
	}

	out := make([]domain.Instance, 0, len(ids))
	for _, id := range ids {
		inst, err := s.repo.LoadInstance(ctx, tx, tenantID, id)
		if err != nil {
			return nil, fmt.Errorf("list pending: load instance %s: %w", id, err)
		}
		out = append(out, *inst)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("list pending: commit: %w", err)
	}
	return out, nil
}
