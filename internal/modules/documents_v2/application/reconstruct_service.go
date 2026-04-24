package application

import (
	"context"
	"database/sql"
	"fmt"

	"metaldocs/internal/modules/iam/authz"
	"metaldocs/internal/modules/render/fanout"
)

type ReconstructionRunner interface {
	Reconstruct(ctx context.Context, tenantID, revisionID string) (fanout.ReconstructionEntry, error)
}

type ReconstructionService struct {
	db     *sql.DB
	runner ReconstructionRunner
}

func NewReconstructionService(db *sql.DB, runner ReconstructionRunner) *ReconstructionService {
	return &ReconstructionService{db: db, runner: runner}
}

func (s *ReconstructionService) GetReconstruction(ctx context.Context, tenantID, actorID, docID string) (fanout.ReconstructionEntry, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fanout.ReconstructionEntry{}, fmt.Errorf("reconstruct authz: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ctx = authz.WithCapCache(ctx)
	if err := setAuthzGUC(ctx, tx, tenantID, actorID); err != nil {
		return fanout.ReconstructionEntry{}, err
	}

	areaCode, err := loadDocumentAreaCode(ctx, tx, tenantID, docID)
	if err != nil {
		return fanout.ReconstructionEntry{}, fmt.Errorf("reconstruct authz: load area: %w", err)
	}
	if err := authz.Require(ctx, tx, "doc.reconstruct", areaCode); err != nil {
		return fanout.ReconstructionEntry{}, err
	}

	entry, err := s.runner.Reconstruct(ctx, tenantID, docID)
	if err != nil {
		return fanout.ReconstructionEntry{}, err
	}
	return entry, nil
}
