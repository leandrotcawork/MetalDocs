package application

import (
	"context"

	"metaldocs/internal/modules/templates_v2/domain"
)

type UpsertApprovalConfigCmd struct {
	TenantID, ActorUserID, TemplateID string
	ActorRoles                        []string
	ReviewerRole                      *string
	ApproverRole                      string
}

func (s *Service) UpsertApprovalConfig(ctx context.Context, cmd UpsertApprovalConfigCmd) (*domain.ApprovalConfig, error) {
	template, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID)
	if err != nil {
		return nil, err
	}
	if template.TenantID != cmd.TenantID {
		return nil, domain.ErrNotFound
	}
	if template.IsArchived() {
		return nil, domain.ErrArchived
	}

	hasEverPublished := template.PublishedVersionID != nil
	if hasEverPublished {
		if !containsRole(cmd.ActorRoles, "admin") {
			return nil, domain.ErrForbidden
		}
	} else {
		if template.CreatedBy != cmd.ActorUserID && !containsRole(cmd.ActorRoles, "admin") {
			return nil, domain.ErrForbidden
		}
	}

	if cmd.ApproverRole == "" {
		return nil, domain.ErrInvalidApprovalConfig
	}

	config := &domain.ApprovalConfig{
		TemplateID:   cmd.TemplateID,
		ReviewerRole: cmd.ReviewerRole,
		ApproverRole: cmd.ApproverRole,
	}

	if err := s.repo.UpsertApprovalConfig(ctx, config); err != nil {
		return nil, err
	}

	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  nil,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditApprovalConfigUpdated,
		Details: map[string]any{
			"reviewer_role": cmd.ReviewerRole,
			"approver_role": cmd.ApproverRole,
		},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return config, nil
}
