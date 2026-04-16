package application

import (
	"context"
	"encoding/json"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

// PublishTemplateForReview transitions a CK5 template draft from draft -> pending_review.
// Returns ErrInvalidTemplateDraftStatus if not in draft state.
// Returns ErrEmptyTemplateContent if the _ck5.contentHtml is empty.
func (s *Service) PublishTemplateForReview(ctx context.Context, key string) error {
	draft, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return domain.ErrTemplateNotFound
	}
	if draft.DraftStatus != domain.TemplateStatusDraft {
		return domain.ErrInvalidTemplateDraftStatus
	}
	html := extractCK5ContentHtml(draft.BlocksJSON)
	if strings.TrimSpace(html) == "" {
		return domain.ErrEmptyTemplateContent
	}
	return s.repo.UpdateTemplateDraftStatus(ctx, key, domain.TemplateStatusPendingReview)
}

// ApproveTemplate transitions a CK5 template draft from pending_review -> published
// and stores the frozen published_html snapshot.
// Returns ErrInvalidTemplateDraftStatus if not in pending_review state.
func (s *Service) ApproveTemplate(ctx context.Context, key string) error {
	draft, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return domain.ErrTemplateNotFound
	}
	if draft.DraftStatus != domain.TemplateStatusPendingReview {
		return domain.ErrInvalidTemplateDraftStatus
	}
	html := extractCK5ContentHtml(draft.BlocksJSON)
	return s.repo.SetTemplateDraftPublished(ctx, key, html)
}

// extractCK5ContentHtml extracts the _ck5.contentHtml value from blocks_json.
// Returns empty string on any parse error or if the key is absent.
func extractCK5ContentHtml(blocksJSON json.RawMessage) string {
	if len(blocksJSON) == 0 {
		return ""
	}
	var wrapper struct {
		CK5 *struct {
			ContentHtml string `json:"contentHtml"`
		} `json:"_ck5"`
	}
	if err := json.Unmarshal(blocksJSON, &wrapper); err != nil {
		return ""
	}
	if wrapper.CK5 == nil {
		return ""
	}
	return wrapper.CK5.ContentHtml
}
