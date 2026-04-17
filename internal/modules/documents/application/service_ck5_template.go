package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

// GetCK5TemplateDraftContent reads CK5 HTML + manifest from a template draft BlocksJSON payload.
func (s *Service) GetCK5TemplateDraftContent(ctx context.Context, templateKey string) (string, map[string]any, error) {
	draft, err := s.repo.GetTemplateDraft(ctx, strings.TrimSpace(templateKey))
	if err != nil {
		return "", nil, err
	}
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateView); err != nil {
		return "", nil, err
	}

	var data map[string]any
	if len(draft.BlocksJSON) > 0 {
		if err := json.Unmarshal(draft.BlocksJSON, &data); err != nil {
			return "", nil, fmt.Errorf("blocks_json corrupt for %q: %w", strings.TrimSpace(templateKey), err)
		}
	}

	ck5Raw, ok := data["_ck5"]
	if !ok || ck5Raw == nil {
		return "", defaultCK5Manifest(), nil
	}
	ck5, ok := ck5Raw.(map[string]any)
	if !ok {
		return "", defaultCK5Manifest(), nil
	}

	html, _ := ck5["contentHtml"].(string)
	manifest, _ := ck5["manifest"].(map[string]any)
	if manifest == nil {
		manifest = defaultCK5Manifest()
	}
	return html, manifest, nil
}

// SaveCK5TemplateDraftAuthorized stores CK5 HTML + manifest under BlocksJSON["_ck5"] with merge semantics.
func (s *Service) SaveCK5TemplateDraftAuthorized(ctx context.Context, templateKey, contentHTML string, manifest map[string]any) error {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return err
	}
	key := strings.TrimSpace(templateKey)
	existing, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return err
	}

	if manifest == nil {
		manifest = defaultCK5Manifest()
	}

	var existingData map[string]any
	if len(existing.BlocksJSON) > 0 {
		if err := json.Unmarshal(existing.BlocksJSON, &existingData); err != nil {
			return fmt.Errorf("blocks_json corrupt for %q: %w", key, err)
		}
	}
	if existingData == nil {
		existingData = map[string]any{}
	}
	existingData["_ck5"] = map[string]any{
		"contentHtml": contentHTML,
		"manifest":    manifest,
	}

	blocksJSON, err := json.Marshal(existingData)
	if err != nil {
		return err
	}

	updated := *existing
	updated.BlocksJSON = blocksJSON

	_, err = s.repo.UpsertTemplateDraftCAS(ctx, &updated, existing.LockVersion)
	return err
}

func defaultCK5Manifest() map[string]any {
	return map[string]any{"fields": []any{}}
}
