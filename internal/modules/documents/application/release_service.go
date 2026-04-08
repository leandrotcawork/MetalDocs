package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type ReleaseInput struct {
	DocumentID string
	DraftID    uuid.UUID
	ApprovedBy string
}

type DocxRenderer interface {
	RenderDocx(ctx context.Context, content []byte) ([]byte, error)
}

type ReleaseRepo interface {
	GetDraft(ctx context.Context, id uuid.UUID) (*draftSnapshot, error)
	ArchivePreviousReleased(ctx context.Context, documentID string) (versionID uuid.UUID, docxBytes []byte, err error)
	PromoteDraftToReleased(ctx context.Context, draftID uuid.UUID, docxBytes []byte, approvedBy string) error
	StoreRevisionDiff(ctx context.Context, versionID uuid.UUID, diff json.RawMessage) error
	DeleteImageRefs(ctx context.Context, versionID uuid.UUID) error
	CleanupOrphanImages(ctx context.Context) error
}

type draftSnapshot struct {
	ID            uuid.UUID
	ContentBlocks []byte
}

type ReleaseService struct {
	repo     ReleaseRepo
	renderer DocxRenderer
}

func NewReleaseService(repo ReleaseRepo, renderer DocxRenderer) *ReleaseService {
	return &ReleaseService{repo: repo, renderer: renderer}
}

// ReleaseDraft executes the atomic release sequence. The actual transaction
// boundary is managed by the repository implementation, which wraps the
// underlying SQL operations in a single BEGIN/COMMIT block.
func (s *ReleaseService) ReleaseDraft(ctx context.Context, in ReleaseInput) error {
	// 1. Render DOCX from draft content (outside transaction; render failures abort early)
	draft, err := s.repo.GetDraft(ctx, in.DraftID)
	if err != nil {
		return err
	}
	docxBytes, err := s.renderer.RenderDocx(ctx, draft.ContentBlocks)
	if err != nil {
		return err
	}

	// 2. Atomic sequence: archive prev → promote draft → store diff → delete refs → orphan cleanup
	prevVersionID, _, err := s.repo.ArchivePreviousReleased(ctx, in.DocumentID)
	if err != nil {
		return err
	}

	if err := s.repo.PromoteDraftToReleased(ctx, in.DraftID, docxBytes, in.ApprovedBy); err != nil {
		return err
	}

	// Diff is computed from canonicalized blocks; here we use a placeholder.
	// Real implementation reads previous canonical content and runs ComputeDiff.
	diffJSON := json.RawMessage(`{"added":[],"removed":[],"modified":[]}`)
	if err := s.repo.StoreRevisionDiff(ctx, in.DraftID, diffJSON); err != nil {
		return err
	}

	if prevVersionID != uuid.Nil {
		if err := s.repo.DeleteImageRefs(ctx, prevVersionID); err != nil {
			return err
		}
	}

	if err := s.repo.CleanupOrphanImages(ctx); err != nil {
		return err
	}

	return nil
}
