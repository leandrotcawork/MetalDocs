package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrSubmitForApprovalDraftNotDraft = errors.New("DOCUMENTS_DRAFT_NOT_DRAFT")
var errSubmitForApprovalServiceNotConfigured = errors.New("submit-for-approval service not configured")

type SubmitForApprovalRepository interface {
	TransitionDraftToPendingApproval(ctx context.Context, draftID uuid.UUID) error
}

type SubmitForApprovalService struct {
	repo SubmitForApprovalRepository
}

func NewSubmitForApprovalService(repo SubmitForApprovalRepository) *SubmitForApprovalService {
	return &SubmitForApprovalService{repo: repo}
}

func (s *SubmitForApprovalService) SubmitForApproval(ctx context.Context, draftID uuid.UUID) error {
	if s == nil || s.repo == nil {
		return errSubmitForApprovalServiceNotConfigured
	}
	if err := s.repo.TransitionDraftToPendingApproval(ctx, draftID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSubmitForApprovalDraftNotDraft
		}
		return fmt.Errorf("submit for approval %s: %w", draftID, err)
	}
	return nil
}
