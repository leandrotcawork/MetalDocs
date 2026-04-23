package application

import (
	"context"
	"fmt"

	"metaldocs/internal/modules/templates_v2/domain"
)

type UpdateSchemasCmd struct {
	TenantID, ActorUserID, TemplateID string
	VersionNumber                     int
	MetadataSchema                    domain.MetadataSchema
	PlaceholderSchema                 []domain.Placeholder
	EditableZones                     []domain.EditableZone
	ExpectedContentHash               string
}

func (s *Service) UpdateSchemas(ctx context.Context, cmd UpdateSchemasCmd) (*domain.TemplateVersion, error) {
	version, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return nil, err
	}
	if version.Status != domain.VersionStatusDraft {
		return nil, domain.ErrInvalidStateTransition
	}
	if cmd.ExpectedContentHash != "" && cmd.ExpectedContentHash != version.ContentHash {
		return nil, domain.ErrStaleBase
	}

	if err := ValidatePlaceholders(cmd.PlaceholderSchema); err != nil {
		return nil, err
	}
	for _, p := range cmd.PlaceholderSchema {
		if p.Type == domain.PHSelect && len(p.Options) == 0 {
			return nil, fmt.Errorf("select_placeholder_requires_options: %s", p.ID)
		}
		if p.Type != domain.PHSelect && len(p.Options) > 0 {
			return nil, fmt.Errorf("options_allowed_only_for_select: %s", p.ID)
		}
	}

	zoneIDs := map[string]struct{}{}
	for _, z := range cmd.EditableZones {
		if _, exists := zoneIDs[z.ID]; exists {
			return nil, fmt.Errorf("duplicate_zone_id: %s", z.ID)
		}
		zoneIDs[z.ID] = struct{}{}
	}

	version.MetadataSchema = cloneMetadataSchema(cmd.MetadataSchema)
	version.PlaceholderSchema = clonePlaceholders(cmd.PlaceholderSchema)
	version.EditableZones = cloneEditableZones(cmd.EditableZones)

	if err := s.repo.UpdateVersion(ctx, version); err != nil {
		return nil, err
	}

	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditSaved,
		Details:    map[string]any{"kind": "schema"},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return version, nil
}

func ValidatePlaceholders(phs []domain.Placeholder) error {
	seen := make(map[string]struct{}, len(phs))
	for i, p := range phs {
		if p.ID == "" {
			return fmt.Errorf("placeholder[%d]: %w", i, domain.ErrPlaceholderIDEmpty)
		}
		if _, exists := seen[p.ID]; exists {
			return fmt.Errorf("duplicate_placeholder_id: %s: %w", p.ID, domain.ErrDuplicatePlaceholderID)
		}
		seen[p.ID] = struct{}{}
	}
	return nil
}
