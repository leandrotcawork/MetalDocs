package application

import (
	"context"
	"errors"
	"fmt"

	"metaldocs/internal/modules/templates_v2/domain"
)

type CreateTemplateCmd struct {
	TenantID      string
	ActorUserID   string
	DocTypeCode   string
	Key           string
	Name          string
	Description   string
	Areas         []string
	Visibility    domain.Visibility
	SpecificAreas []string
	ApproverRole  string
	ReviewerRole  *string
}

type CreateTemplateResult struct {
	Template *domain.Template
	Version  *domain.TemplateVersion
}

func (s *Service) CreateTemplate(ctx context.Context, cmd CreateTemplateCmd) (*CreateTemplateResult, error) {
	if !isValidVisibility(cmd.Visibility) {
		return nil, domain.ErrInvalidVisibility
	}
	if cmd.Visibility == domain.VisibilitySpecific && len(cmd.SpecificAreas) == 0 {
		return nil, fmt.Errorf("%w: specific_visibility_requires_areas", domain.ErrInvalidVisibility)
	}
	if cmd.Visibility != domain.VisibilitySpecific && len(cmd.SpecificAreas) > 0 {
		cmd.SpecificAreas = nil
	}

	if _, err := s.repo.GetTemplateByKey(ctx, cmd.TenantID, cmd.Key); !errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrKeyConflict
	}

	template := &domain.Template{
		ID:                 s.uuid.New(),
		TenantID:           cmd.TenantID,
		DocTypeCode:        cmd.DocTypeCode,
		Key:                cmd.Key,
		Name:               cmd.Name,
		Description:        cmd.Description,
		Areas:              append([]string{}, cmd.Areas...),
		Visibility:         cmd.Visibility,
		SpecificAreas:      append([]string{}, cmd.SpecificAreas...),
		LatestVersion:      1,
		PublishedVersionID: nil,
		CreatedBy:          cmd.ActorUserID,
		CreatedAt:          s.clock.Now(),
	}

	version := &domain.TemplateVersion{
		ID:                  s.uuid.New(),
		TemplateID:          template.ID,
		VersionNumber:       1,
		Status:              domain.VersionStatusDraft,
		DocxStorageKey:      fmt.Sprintf("templates/%s/versions/1.docx", template.ID),
		ContentHash:         "",
		MetadataSchema:      domain.MetadataSchema{},
		PlaceholderSchema:   []domain.Placeholder{},
		EditableZones:       []domain.EditableZone{},
		AuthorID:            cmd.ActorUserID,
		PendingApproverRole: cmd.ApproverRole,
		PendingReviewerRole: cmd.ReviewerRole,
		CreatedAt:           s.clock.Now(),
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, err
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		return nil, err
	}

	if err := s.repo.UpsertApprovalConfig(ctx, &domain.ApprovalConfig{
		TemplateID:   template.ID,
		ApproverRole: cmd.ApproverRole,
		ReviewerRole: cmd.ReviewerRole,
	}); err != nil {
		return nil, err
	}

	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: template.ID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditCreated,
		Details:    map[string]any{},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return &CreateTemplateResult{
		Template: template,
		Version:  version,
	}, nil
}

type CreateVersionCmd struct {
	TenantID    string
	ActorUserID string
	TemplateID  string
}

func (s *Service) CreateNextVersion(ctx context.Context, cmd CreateVersionCmd) (*domain.TemplateVersion, error) {
	template, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID)
	if err != nil {
		return nil, err
	}
	if template.IsArchived() {
		return nil, domain.ErrArchived
	}

	var source *domain.TemplateVersion
	if template.PublishedVersionID != nil {
		source, err = s.repo.GetVersionByID(ctx, *template.PublishedVersionID)
	} else {
		source, err = s.repo.GetVersion(ctx, template.ID, template.LatestVersion)
	}
	if err != nil {
		return nil, err
	}

	newNum := template.LatestVersion + 1
	version := &domain.TemplateVersion{
		ID:                s.uuid.New(),
		TemplateID:        cmd.TemplateID,
		VersionNumber:     newNum,
		Status:            domain.VersionStatusDraft,
		DocxStorageKey:    fmt.Sprintf("templates/%s/versions/%d.docx", cmd.TemplateID, newNum),
		ContentHash:       "",
		MetadataSchema:    cloneMetadataSchema(source.MetadataSchema),
		PlaceholderSchema: clonePlaceholders(source.PlaceholderSchema),
		EditableZones:     cloneEditableZones(source.EditableZones),
		AuthorID:          cmd.ActorUserID,
		CreatedAt:         s.clock.Now(),
	}

	if err := s.repo.CreateVersion(ctx, version); err != nil {
		return nil, err
	}

	template.LatestVersion = newNum
	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, err
	}

	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: template.ID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditCreated,
		Details:    map[string]any{},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return version, nil
}

func isValidVisibility(v domain.Visibility) bool {
	return v == domain.VisibilityPublic || v == domain.VisibilityInternal || v == domain.VisibilitySpecific
}

func cloneMetadataSchema(s domain.MetadataSchema) domain.MetadataSchema {
	return domain.MetadataSchema{
		DocCodePattern:      s.DocCodePattern,
		RetentionDays:       s.RetentionDays,
		DistributionDefault: cloneStringSlice(s.DistributionDefault),
		RequiredMetadata:    cloneStringSlice(s.RequiredMetadata),
	}
}

func clonePlaceholders(in []domain.Placeholder) []domain.Placeholder {
	out := make([]domain.Placeholder, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Options = cloneStringSlice(in[i].Options)
	}
	return out
}

func cloneEditableZones(in []domain.EditableZone) []domain.EditableZone {
	out := make([]domain.EditableZone, len(in))
	copy(out, in)
	return out
}

func cloneStringSlice(in []string) []string {
	if in == nil {
		return nil
	}
	return append([]string{}, in...)
}
