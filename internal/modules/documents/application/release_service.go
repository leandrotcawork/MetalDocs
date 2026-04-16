package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain"
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
	GetDraft(ctx context.Context, id uuid.UUID) (*DraftSnapshot, error)
	ArchivePreviousReleased(ctx context.Context, documentID string) (versionID uuid.UUID, prevContentBlocks []byte, docxBytes []byte, err error)
	PromoteDraftToReleased(ctx context.Context, draftID uuid.UUID, docxBytes []byte, approvedBy string) error
	StoreRevisionDiff(ctx context.Context, versionID uuid.UUID, diff json.RawMessage) error
	DeleteImageRefs(ctx context.Context, versionID uuid.UUID) error
	CleanupOrphanImages(ctx context.Context) error
}

// DraftSnapshot carries the data needed to execute the release sequence,
// including version metadata required by the renderer pin capturer.
type DraftSnapshot struct {
	ID            uuid.UUID
	ContentBlocks []byte

	// Version metadata — populated by GetDraft so the release service can
	// build a domain.Version for the renderer pin capturer.
	DocumentID      string
	VersionNumber   int
	ContentSource   string
	TemplateKey     string
	TemplateVersion int
}

type ReleaseService struct {
	repo                ReleaseRepo
	renderer            DocxRenderer
	rendererPinCapturer *RendererPinCapturer
}

func NewReleaseService(repo ReleaseRepo, renderer DocxRenderer) *ReleaseService {
	return &ReleaseService{repo: repo, renderer: renderer}
}

func (s *ReleaseService) WithRendererPinCapturer(c *RendererPinCapturer) *ReleaseService {
	s.rendererPinCapturer = c
	return s
}

// ReleaseDraft executes the atomic release sequence. The actual transaction
// boundary is managed by the repository implementation, which wraps the
// underlying SQL operations in a single BEGIN/COMMIT block.
//
// The renderer pin is captured BEFORE the status write. If capture fails the
// release is aborted (fail-loud). An observed RELEASED status in the database
// always implies a valid pin for browser_editor content. If any downstream
// step fails AFTER the pin has been captured, a compensating rollback clears
// the pin so draft rows never retain a stale pin.
func (s *ReleaseService) ReleaseDraft(ctx context.Context, in ReleaseInput) (retErr error) {
	// 1. Render DOCX from draft content (outside transaction; render failures abort early)
	draft, err := s.repo.GetDraft(ctx, in.DraftID)
	if err != nil {
		return err
	}
	docxBytes, err := s.renderer.RenderDocx(ctx, draft.ContentBlocks)
	if err != nil {
		return err
	}

	// 2. Capture renderer pin BEFORE committing the release status.
	//    For non-browser-editor sources OnRelease is a no-op.
	var pinnedVersion *domain.Version
	if s.rendererPinCapturer != nil {
		version := domain.Version{
			DocumentID:      draft.DocumentID,
			Number:          draft.VersionNumber,
			ContentSource:   draft.ContentSource,
			TemplateKey:     draft.TemplateKey,
			TemplateVersion: draft.TemplateVersion,
		}
		if err := s.rendererPinCapturer.OnRelease(ctx, version); err != nil {
			return fmt.Errorf("capture renderer pin before release: %w", err)
		}
		pinnedVersion = &version
		defer func() {
			// Compensate on ANY post-capture failure: clear the pin so the draft
			// row is not left with renderer_pin populated. Rollback is a no-op
			// for non-browser-editor sources, so native/docx_upload paths pay no
			// cost. The rollback best-effort: if it also fails, we surface both
			// errors so ops can see the split-brain state.
			if retErr == nil {
				return
			}
			if rbErr := s.rendererPinCapturer.Rollback(context.WithoutCancel(ctx), *pinnedVersion); rbErr != nil {
				retErr = fmt.Errorf("%w (pin rollback also failed: %v)", retErr, rbErr)
			}
		}()
	}

	// 3. Atomic sequence: archive prev → promote draft → store diff → delete refs → orphan cleanup
	prevVersionID, prevContentBlocks, _, err := s.repo.ArchivePreviousReleased(ctx, in.DocumentID)
	if err != nil {
		return err
	}

	if err := s.repo.PromoteDraftToReleased(ctx, in.DraftID, docxBytes, in.ApprovedBy); err != nil {
		return err
	}

	// Compute real diff from previous and current canonical content blocks.
	var prevBlocks []any
	if len(prevContentBlocks) > 0 {
		var prevEnvelope map[string]any
		if err := json.Unmarshal(prevContentBlocks, &prevEnvelope); err == nil {
			prevBlocks, _ = prevEnvelope["blocks"].([]any)
		}
	}
	var currBlocks []any
	var currEnvelope map[string]any
	if err := json.Unmarshal(draft.ContentBlocks, &currEnvelope); err == nil {
		currBlocks, _ = currEnvelope["blocks"].([]any)
	}
	diff := computeRevisionDiff(prevBlocks, currBlocks)
	diffJSON, _ := json.Marshal(diff)
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

func computeRevisionDiff(prevBlocks, currBlocks []any) map[string][]any {
	// Keep persisted shape stable after removing the old mddm package.
	if jsonEqualLoose(prevBlocks, currBlocks) {
		return map[string][]any{
			"added":    []any{},
			"removed":  []any{},
			"modified": []any{},
		}
	}
	return map[string][]any{
		"added":    currBlocks,
		"removed":  prevBlocks,
		"modified": []any{},
	}
}

func jsonEqualLoose(a, b []any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
