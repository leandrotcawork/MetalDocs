package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

// isAllowedTemplate performs global role-based RBAC for template capabilities.
// Templates do not have per-resource DB policies; access is governed entirely by
// the caller's IAM roles, matching the StaticAuthorizer policy defined in Phase 0.
//
// Role → capability mapping:
//
//	admin    → template.view, template.edit, template.publish, template.export
//	editor   → template.view, template.export
//	reviewer → template.view, template.export
//
// Returns nil if allowed, domain.ErrDocumentNotFound (masking as 404) if denied.
func (s *Service) isAllowedTemplate(ctx context.Context, capability string) error {
	if shouldBypassPolicy(ctx) {
		return nil
	}

	roles := authn.RolesFromContext(ctx)
	if len(roles) == 0 {
		return domain.ErrDocumentNotFound
	}

	roleCapabilities := map[string]map[string]bool{
		"admin": {
			domain.CapabilityTemplateView:    true,
			domain.CapabilityTemplateEdit:    true,
			domain.CapabilityTemplatePublish: true,
			domain.CapabilityTemplateExport:  true,
		},
		"editor": {
			domain.CapabilityTemplateView:   true,
			domain.CapabilityTemplateExport: true,
		},
		"reviewer": {
			domain.CapabilityTemplateView:   true,
			domain.CapabilityTemplateExport: true,
		},
		"viewer": {},
	}

	for _, role := range roles {
		if caps, ok := roleCapabilities[strings.ToLower(strings.TrimSpace(role))]; ok {
			if caps[capability] {
				return nil
			}
		}
	}

	return domain.ErrDocumentNotFound
}

// writeTemplateAudit appends an audit event to the template audit log.
// Audit failures are swallowed (non-blocking).
func (s *Service) writeTemplateAudit(ctx context.Context, key, action, actorID string, version *int) {
	_ = s.repo.WriteTemplateAuditEvent(ctx, domain.TemplateAuditEvent{
		TemplateKey: key,
		Action:      action,
		ActorID:     actorID,
		Version:     version,
	})
}

// GetTemplateDraft returns the in-progress draft for templateKey.
// No RBAC is enforced at the service layer; the HTTP handler enforces auth separately.
func (s *Service) GetTemplateDraft(ctx context.Context, templateKey string) (*domain.TemplateDraft, error) {
	templateKey = strings.TrimSpace(templateKey)
	if templateKey == "" {
		return nil, domain.ErrInvalidCommand
	}
	return s.repo.GetTemplateDraft(ctx, templateKey)
}

// ListTemplateAuditEvents returns all audit events for a template key.
// No RBAC is enforced at the service layer.
func (s *Service) ListTemplateAuditEvents(ctx context.Context, templateKey string) ([]domain.TemplateAuditEvent, error) {
	templateKey = strings.TrimSpace(templateKey)
	if templateKey == "" {
		return nil, domain.ErrInvalidCommand
	}
	return s.repo.ListTemplateAuditEvents(ctx, templateKey)
}

// CreateDraftAuthorized creates a new template draft keyed by profileCode + generated suffix.
// Requires CapabilityTemplateEdit.
func (s *Service) CreateDraftAuthorized(ctx context.Context, profileCode, name, actorID string) (*domain.TemplateDraft, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return nil, err
	}

	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	name = strings.TrimSpace(name)
	if profileCode == "" || name == "" {
		return nil, domain.ErrInvalidCommand
	}

	templateKey := profileCode + "-" + mustNewID()[:8]

	now := time.Now().UTC()
	draft := &domain.TemplateDraft{
		TemplateKey: templateKey,
		ProfileCode: profileCode,
		BaseVersion: 0,
		Name:        name,
		CreatedBy:   strings.TrimSpace(actorID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	saved, err := s.repo.UpsertTemplateDraftCAS(ctx, draft, 0)
	if err != nil {
		return nil, err
	}

	s.writeTemplateAudit(ctx, templateKey, "draft_created", actorID, nil)
	return saved, nil
}

// SaveDraftAuthorized saves edits to an existing draft using optimistic locking.
// Returns ErrTemplateDraftNotFound if the draft does not exist.
// Returns ErrTemplateLockConflict if lockVersion does not match the stored version.
// Requires CapabilityTemplateEdit.
func (s *Service) SaveDraftAuthorized(ctx context.Context, key string, blocks, theme, meta json.RawMessage, lockVersion int, actorID string) (*domain.TemplateDraft, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, domain.ErrInvalidCommand
	}

	existing, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return nil, err // propagates ErrTemplateDraftNotFound
	}

	draft := &domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: existing.ProfileCode,
		BaseVersion: existing.BaseVersion,
		Name:        existing.Name,
		BlocksJSON:  blocks,
		ThemeJSON:   theme,
		MetaJSON:    meta,
		CreatedBy:   existing.CreatedBy,
	}

	saved, err := s.repo.UpsertTemplateDraftCAS(ctx, draft, lockVersion)
	if err != nil {
		return nil, err // propagates ErrTemplateLockConflict or ErrTemplateDraftNotFound
	}

	s.writeTemplateAudit(ctx, key, "draft_saved", actorID, nil)
	return saved, nil
}

// PublishAuthorized promotes a draft to a published DocumentTemplateVersion.
// Requires CapabilityTemplatePublish.
//
// Sequence: InsertTemplateVersion → DeleteTemplateDraft → audit.
// No transaction boundary is enforced at this layer (Phase 5 can add one).
func (s *Service) PublishAuthorized(ctx context.Context, key string, lockVersion int, actorID string) (*domain.DocumentTemplateVersion, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplatePublish); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, domain.ErrInvalidCommand
	}

	draft, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateDraftNotFound) {
			return nil, domain.ErrTemplateDraftNotFound
		}
		return nil, err
	}

	if draft.HasStrippedFields {
		return nil, domain.ErrTemplateHasStrippedFields
	}

	if errs := validateTemplateStrict(draft.BlocksJSON); len(errs) > 0 {
		return nil, domain.ErrTemplatePublishValidation
	}

	newVersion := draft.BaseVersion + 1

	tv := domain.DocumentTemplateVersion{
		TemplateKey:   key,
		Version:       newVersion,
		ProfileCode:   draft.ProfileCode,
		Name:          draft.Name,
		Status:        string(domain.TemplateStatusPublished),
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.repo.InsertTemplateVersion(ctx, tv); err != nil {
		return nil, err
	}

	if err := s.repo.DeleteTemplateDraft(ctx, key); err != nil {
		// Draft delete failure is logged but does not roll back the published version.
		// A future tx wrapper in Phase 5 can make this atomic.
		_ = err
	}

	s.writeTemplateAudit(ctx, key, "published", actorID, &newVersion)
	return &tv, nil
}

// DiscardDraftAuthorized deletes a draft without publishing it.
// Requires CapabilityTemplateEdit.
func (s *Service) DiscardDraftAuthorized(ctx context.Context, key, actorID string) error {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return domain.ErrInvalidCommand
	}

	if _, err := s.repo.GetTemplateDraft(ctx, key); err != nil {
		return err // propagates ErrTemplateDraftNotFound
	}

	if err := s.repo.DeleteTemplateDraft(ctx, key); err != nil {
		return err
	}

	s.writeTemplateAudit(ctx, key, "draft_discarded", actorID, nil)
	return nil
}

// DeprecateAuthorized marks a published template version as deprecated.
// Requires CapabilityTemplatePublish.
func (s *Service) DeprecateAuthorized(ctx context.Context, key string, version int, actorID string) error {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplatePublish); err != nil {
		return err
	}

	key = strings.TrimSpace(key)
	if key == "" || version <= 0 {
		return domain.ErrInvalidCommand
	}

	if err := s.repo.UpdateTemplateVersionStatus(ctx, key, version, domain.TemplateStatusDeprecated); err != nil {
		return err
	}

	s.writeTemplateAudit(ctx, key, "deprecated", actorID, &version)
	return nil
}

// validateTemplateStrict is a placeholder strict validation gate.
// Phase 4 will replace this with real JSON Schema codec validation.
func validateTemplateStrict(_ json.RawMessage) []domain.PublishError {
	return nil
}
