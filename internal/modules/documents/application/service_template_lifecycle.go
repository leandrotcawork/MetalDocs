package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/domain/mddm"
	"metaldocs/internal/platform/authn"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ListTemplatesByProfile returns all template versions for a profile code.
// No RBAC enforced at service layer; the HTTP handler enforces auth separately.
func (s *Service) ListTemplatesByProfile(ctx context.Context, profileCode string) ([]domain.DocumentTemplateVersion, error) {
	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	if profileCode == "" {
		return nil, domain.ErrInvalidCommand
	}
	return s.repo.ListDocumentTemplateVersions(ctx, profileCode)
}

// GetLatestPublishedTemplate returns the highest-versioned published template for the given key.
// No RBAC enforced at service layer.
func (s *Service) GetLatestPublishedTemplate(ctx context.Context, key string) (domain.DocumentTemplateVersion, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return domain.DocumentTemplateVersion{}, domain.ErrInvalidCommand
	}

	allVersions, err := s.repo.ListDocumentTemplateVersions(ctx, "")
	if err != nil {
		return domain.DocumentTemplateVersion{}, err
	}

	var latest *domain.DocumentTemplateVersion
	for i := range allVersions {
		v := &allVersions[i]
		if v.TemplateKey == key {
			if latest == nil || v.Version > latest.Version {
				latest = v
			}
		}
	}

	if latest == nil {
		return domain.DocumentTemplateVersion{}, domain.ErrTemplateNotFound
	}

	return *latest, nil
}

// GetTemplateVersion returns a specific published template version.
// Maps ErrDocumentTemplateNotFound to ErrTemplateNotFound.
func (s *Service) GetTemplateVersion(ctx context.Context, key string, version int) (domain.DocumentTemplateVersion, error) {
	key = strings.TrimSpace(key)
	if key == "" || version <= 0 {
		return domain.DocumentTemplateVersion{}, domain.ErrInvalidCommand
	}
	tv, err := s.repo.GetDocumentTemplateVersion(ctx, key, version)
	if errors.Is(err, domain.ErrDocumentTemplateNotFound) {
		return domain.DocumentTemplateVersion{}, domain.ErrTemplateNotFound
	}
	return tv, err
}

// AcknowledgeStrippedFieldsAuthorized clears the HasStrippedFields flag so publishing can proceed.
// Requires CapabilityTemplateEdit.
func (s *Service) AcknowledgeStrippedFieldsAuthorized(ctx context.Context, key string, lockVersion int, actorID string) (*domain.TemplateDraft, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, domain.ErrInvalidCommand
	}

	draft, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return nil, err
	}

	if draft.LockVersion != lockVersion {
		return nil, domain.ErrTemplateLockConflict
	}

	draft.HasStrippedFields = false
	draft.StrippedFieldsJSON = nil

	saved, err := s.repo.UpsertTemplateDraftCAS(ctx, draft, lockVersion)
	if err != nil {
		return nil, err
	}

	s.writeTemplateAudit(ctx, key, "stripped_fields_acknowledged", actorID, nil)
	return saved, nil
}

// EditPublishedAuthorized creates a draft from the latest published version (idempotent if draft exists).
// Requires CapabilityTemplateEdit.
func (s *Service) EditPublishedAuthorized(ctx context.Context, key string, actorID string) (*domain.TemplateDraft, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, domain.ErrInvalidCommand
	}

	// Idempotent: return existing draft if one already exists.
	existing, err := s.repo.GetTemplateDraft(ctx, key)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrTemplateDraftNotFound) {
		return nil, err
	}

	// Find the latest published version by listing all versions for the template key.
	// ListDocumentTemplateVersions groups by profileCode so we pass "" to get all,
	// then filter by key manually.
	allVersions, err := s.repo.ListDocumentTemplateVersions(ctx, "")
	if err != nil {
		return nil, err
	}

	var latestVersion *domain.DocumentTemplateVersion
	for i := range allVersions {
		v := &allVersions[i]
		if v.TemplateKey == key {
			if latestVersion == nil || v.Version > latestVersion.Version {
				latestVersion = v
			}
		}
	}
	if latestVersion == nil {
		return nil, domain.ErrTemplateNotFound
	}

	// Marshal Definition back to JSON for BlocksJSON.
	var blocksJSON json.RawMessage
	if latestVersion.Definition != nil {
		b, merr := json.Marshal(latestVersion.Definition)
		if merr != nil {
			return nil, merr
		}
		blocksJSON = json.RawMessage(b)
	} else if latestVersion.Body != "" {
		blocksJSON = json.RawMessage(latestVersion.Body)
	}

	now := time.Now().UTC()
	draft := &domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: latestVersion.ProfileCode,
		BaseVersion: latestVersion.Version,
		Name:        latestVersion.Name,
		BlocksJSON:  blocksJSON,
		CreatedBy:   strings.TrimSpace(actorID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	saved, err := s.repo.UpsertTemplateDraftCAS(ctx, draft, 0)
	if err != nil {
		return nil, err
	}

	s.writeTemplateAudit(ctx, key, "edit_published_started", actorID, nil)
	return saved, nil
}

// CloneAuthorized creates a new draft as a copy of an existing draft or published template.
// Requires CapabilityTemplateEdit.
func (s *Service) CloneAuthorized(ctx context.Context, sourceKey, newName string, actorID string) (*domain.TemplateDraft, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return nil, err
	}

	sourceKey = strings.TrimSpace(sourceKey)
	newName = strings.TrimSpace(newName)
	if sourceKey == "" {
		return nil, domain.ErrInvalidCommand
	}

	// Try draft first, then fall back to latest published version.
	var profileCode string
	var blocksJSON json.RawMessage

	srcDraft, draftErr := s.repo.GetTemplateDraft(ctx, sourceKey)
	if draftErr == nil {
		profileCode = srcDraft.ProfileCode
		blocksJSON = srcDraft.BlocksJSON
	} else {
		// Fall back to latest published version.
		allVersions, err := s.repo.ListDocumentTemplateVersions(ctx, "")
		if err != nil {
			return nil, err
		}
		var latestVersion *domain.DocumentTemplateVersion
		for i := range allVersions {
			v := &allVersions[i]
			if v.TemplateKey == sourceKey {
				if latestVersion == nil || v.Version > latestVersion.Version {
					latestVersion = v
				}
			}
		}
		if latestVersion == nil {
			return nil, domain.ErrTemplateNotFound
		}
		profileCode = latestVersion.ProfileCode
		if latestVersion.Definition != nil {
			b, merr := json.Marshal(latestVersion.Definition)
			if merr != nil {
				return nil, merr
			}
			blocksJSON = json.RawMessage(b)
		} else if latestVersion.Body != "" {
			blocksJSON = json.RawMessage(latestVersion.Body)
		}
	}

	if newName == "" {
		newName = sourceKey + " (copy)"
	}

	newKey := sourceKey + "-copy-" + mustNewID()[:6]
	now := time.Now().UTC()
	draft := &domain.TemplateDraft{
		TemplateKey: newKey,
		ProfileCode: profileCode,
		BaseVersion: 0,
		Name:        newName,
		BlocksJSON:  blocksJSON,
		CreatedBy:   strings.TrimSpace(actorID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	saved, err := s.repo.UpsertTemplateDraftCAS(ctx, draft, 0)
	if err != nil {
		return nil, err
	}

	s.writeTemplateAudit(ctx, newKey, "cloned", actorID, nil)
	return saved, nil
}

// DeleteDraftAuthorized permanently deletes a draft without publishing.
// Requires CapabilityTemplateEdit.
func (s *Service) DeleteDraftAuthorized(ctx context.Context, key, actorID string) error {
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

	s.writeTemplateAudit(ctx, key, "draft_deleted", actorID, nil)
	return nil
}

// ExportTemplate serializes a published template version for download/transfer.
// Requires CapabilityTemplateExport.
func (s *Service) ExportTemplate(ctx context.Context, key string, version int, actorID string) ([]byte, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateExport); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" || version <= 0 {
		return nil, domain.ErrInvalidCommand
	}

	tv, err := s.repo.GetDocumentTemplateVersion(ctx, key, version)
	if errors.Is(err, domain.ErrDocumentTemplateNotFound) {
		return nil, domain.ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}

	export := map[string]any{
		"templateKey": tv.TemplateKey,
		"version":     tv.Version,
		"profileCode": tv.ProfileCode,
		"name":        tv.Name,
		"definition":  tv.Definition,
	}

	data, err := json.Marshal(export)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ImportTemplateAuthorized creates a draft from imported JSON template data.
// Requires CapabilityTemplateEdit.
func (s *Service) ImportTemplateAuthorized(ctx context.Context, profileCode string, data []byte, actorID string) (*domain.TemplateDraft, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateEdit); err != nil {
		return nil, err
	}

	profileCode = strings.ToLower(strings.TrimSpace(profileCode))
	if profileCode == "" {
		return nil, domain.ErrInvalidCommand
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, domain.ErrInvalidCommand
	}

	var blocksJSON json.RawMessage
	if def, ok := payload["definition"]; ok {
		blocksJSON = def
	} else if blocks, ok := payload["blocks"]; ok {
		blocksJSON = blocks
	}

	name := profileCode + " imported"
	if rawName, ok := payload["name"]; ok {
		var n string
		if err := json.Unmarshal(rawName, &n); err == nil && n != "" {
			name = n
		}
	}

	templateKey := profileCode + "-" + mustNewID()[:8]
	now := time.Now().UTC()
	draft := &domain.TemplateDraft{
		TemplateKey:       templateKey,
		ProfileCode:       profileCode,
		BaseVersion:       0,
		Name:              name,
		BlocksJSON:        blocksJSON,
		HasStrippedFields: false,
		CreatedBy:         strings.TrimSpace(actorID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	saved, err := s.repo.UpsertTemplateDraftCAS(ctx, draft, 0)
	if err != nil {
		return nil, err
	}

	s.writeTemplateAudit(ctx, templateKey, "imported", actorID, nil)
	return saved, nil
}

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

	if draft.LockVersion != lockVersion {
		return nil, domain.ErrTemplateLockConflict
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

// validateTemplateStrict runs JSON Schema + Layer 2 business-rule validation
// against a blocks-only JSON array (as stored in TemplateDraft.BlocksJSON).
// It synthesizes a minimal envelope so ValidateMDDMBytes can validate the full document.
func validateTemplateStrict(blocksJSON json.RawMessage) []domain.PublishError {
	// Treat nil/empty blocks as an empty array — an empty template is schema-valid.
	blocks := blocksJSON
	if len(blocks) == 0 {
		blocks = json.RawMessage(`[]`)
	}

	// 1. Synthesize a minimal full envelope (schema requires mddm_version + template_ref + blocks).
	envelope := fmt.Sprintf(`{"mddm_version":1,"template_ref":null,"blocks":%s}`, string(blocks))

	// 2. JSON Schema validation.
	if err := mddm.ValidateMDDMBytes([]byte(envelope)); err != nil {
		var verr *jsonschema.ValidationError
		if errors.As(err, &verr) {
			// Walk all leaf causes for field-level detail.
			leaves := collectLeafErrors(verr)
			if len(leaves) == 0 {
				leaves = []domain.PublishError{{Reason: verr.Error()}}
			}
			return leaves
		}
		// JSON parse or other error.
		return []domain.PublishError{{Reason: err.Error()}}
	}

	// 3. Business-rule (Layer 2) validation.
	// Parse the envelope into map[string]any for EnforceLayer2.
	var envelopeMap map[string]any
	if err := json.Unmarshal([]byte(envelope), &envelopeMap); err != nil {
		return []domain.PublishError{{Reason: fmt.Sprintf("envelope parse: %s", err.Error())}}
	}

	// Minimal RulesContext: no DB checkers needed for template validation
	// (image auth + cross-doc refs are skipped when checkers are nil).
	rctx := mddm.RulesContext{}
	if err := mddm.EnforceLayer2(rctx, envelopeMap); err != nil {
		var rv *mddm.RuleViolation
		if errors.As(err, &rv) {
			return []domain.PublishError{{
				BlockID: rv.BlockID,
				Reason:  fmt.Sprintf("[%s] %s", rv.Code, rv.Message),
			}}
		}
		return []domain.PublishError{{Reason: err.Error()}}
	}

	return nil
}

// PreviewDocxAuthorized renders a .docx preview from the current draft's blocks
// without persisting anything. Requires CapabilityTemplateView.
// Returns raw DOCX bytes on success or an error.
func (s *Service) PreviewDocxAuthorized(ctx context.Context, key, actorID string) ([]byte, error) {
	if err := s.isAllowedTemplate(ctx, domain.CapabilityTemplateView); err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, domain.ErrInvalidCommand
	}

	draft, err := s.repo.GetTemplateDraft(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(draft.BlocksJSON) == 0 {
		return nil, fmt.Errorf("preview docx: draft has no blocks")
	}

	// Build synthetic document and version so the existing MDDM DOCX pipeline
	// can render the template blocks as if they were a filled document body.
	// The Envelope the docgen service expects is exactly the BlocksJSON stored
	// in the draft (same {blocks:[...]} format used by document versions).
	syntheticDoc := domain.Document{
		DocumentCode:    key,
		Title:           draft.Name,
		DocumentProfile: draft.ProfileCode,
		Status:          "preview",
	}
	syntheticVersion := domain.Version{
		Number:  0,
		Content: string(draft.BlocksJSON),
	}

	docxBytes, err := s.generateBrowserDocxBytesWithTemplate(ctx, syntheticDoc, syntheticVersion, nil, nil, actorID)
	if err != nil {
		return nil, fmt.Errorf("preview docx: %w", err)
	}

	return docxBytes, nil
}

// collectLeafErrors recursively walks a ValidationError tree and returns
// PublishError entries for each leaf node (errors with no further Causes).
func collectLeafErrors(verr *jsonschema.ValidationError) []domain.PublishError {
	if len(verr.Causes) == 0 {
		loc := strings.Join(verr.InstanceLocation, "/")
		reason := verr.Error()
		return []domain.PublishError{{Field: loc, Reason: reason}}
	}
	var out []domain.PublishError
	for _, cause := range verr.Causes {
		out = append(out, collectLeafErrors(cause)...)
	}
	return out
}
