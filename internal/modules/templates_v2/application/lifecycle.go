package application

import (
	"context"

	"metaldocs/internal/modules/templates_v2/domain"
)

type SubmitForReviewCmd struct {
	TenantID, ActorUserID, TemplateID string
	VersionNumber                     int
}

func (s *Service) SubmitForReview(ctx context.Context, cmd SubmitForReviewCmd) (*domain.TemplateVersion, error) {
	if _, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID); err != nil {
		return nil, err
	}

	version, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return nil, err
	}
	if version.Status != domain.VersionStatusDraft {
		return nil, domain.ErrInvalidStateTransition
	}

	config, err := s.repo.GetApprovalConfig(ctx, cmd.TemplateID)
	if err != nil {
		return nil, err
	}

	version.PendingReviewerRole = config.ReviewerRole
	version.PendingApproverRole = config.ApproverRole
	if err := version.CanTransition(domain.VersionStatusInReview, config.HasReviewer()); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	version.Status = domain.VersionStatusInReview
	version.SubmittedAt = &now

	if err := s.repo.UpdateVersion(ctx, version); err != nil {
		return nil, err
	}
	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditSubmitted,
		Details: map[string]any{
			"reviewer_role": config.ReviewerRole,
			"approver_role": config.ApproverRole,
		},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return version, nil
}

type ReviewCmd struct {
	TenantID, ActorUserID string
	ActorRoles            []string
	TemplateID            string
	VersionNumber         int
	Accept                bool
	Reason                string
}

func (s *Service) Review(ctx context.Context, cmd ReviewCmd) (*domain.TemplateVersion, error) {
	if _, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID); err != nil {
		return nil, err
	}

	version, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return nil, err
	}
	if version.Status != domain.VersionStatusInReview {
		return nil, domain.ErrInvalidStateTransition
	}
	if version.PendingReviewerRole == nil {
		return nil, domain.ErrInvalidStateTransition
	}
	if !containsRole(cmd.ActorRoles, *version.PendingReviewerRole) {
		return nil, domain.ErrForbiddenRole
	}
	if err := domain.CheckSegregation("reviewer", cmd.ActorUserID, version.AuthorID, nil); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if cmd.Accept {
		if err := version.CanTransition(domain.VersionStatusApproved, true); err != nil {
			return nil, err
		}
		version.Status = domain.VersionStatusApproved
		version.ReviewerID = &cmd.ActorUserID
		version.ReviewedAt = &now

		if err := s.repo.UpdateVersion(ctx, version); err != nil {
			return nil, err
		}
		if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
			TenantID:   cmd.TenantID,
			TemplateID: cmd.TemplateID,
			VersionID:  &version.ID,
			ActorID:    cmd.ActorUserID,
			Action:     domain.AuditReviewed,
			Details:    map[string]any{},
			OccurredAt: s.clock.Now(),
		}); err != nil {
			return nil, err
		}
		return version, nil
	}

	if err := version.CanTransition(domain.VersionStatusDraft, true); err != nil {
		return nil, err
	}
	version.Status = domain.VersionStatusDraft
	version.SubmittedAt = nil

	if err := s.repo.UpdateVersion(ctx, version); err != nil {
		return nil, err
	}
	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditRejected,
		Details: map[string]any{
			"reason": cmd.Reason,
			"stage":  "reviewer",
		},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return version, nil
}

type ApproveCmd struct {
	TenantID, ActorUserID string
	ActorRoles            []string
	TemplateID            string
	VersionNumber         int
	Accept                bool
	Reason                string
}

func (s *Service) Approve(ctx context.Context, cmd ApproveCmd) (*domain.TemplateVersion, error) {
	template, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID)
	if err != nil {
		return nil, err
	}
	version, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return nil, err
	}

	hasReviewer := version.PendingReviewerRole != nil
	if hasReviewer {
		if version.Status != domain.VersionStatusApproved {
			return nil, domain.ErrInvalidStateTransition
		}
	} else if version.Status != domain.VersionStatusInReview {
		return nil, domain.ErrInvalidStateTransition
	}

	if !containsRole(cmd.ActorRoles, version.PendingApproverRole) {
		return nil, domain.ErrForbiddenRole
	}
	if err := domain.CheckSegregation("approver", cmd.ActorUserID, version.AuthorID, version.ReviewerID); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	if cmd.Accept {
		if err := version.CanTransition(domain.VersionStatusPublished, hasReviewer); err != nil {
			return nil, err
		}
		version.Status = domain.VersionStatusPublished
		version.ApproverID = &cmd.ActorUserID
		version.ApprovedAt = &now
		version.PublishedAt = &now

		if err := s.repo.ObsoletePreviousPublished(ctx, cmd.TemplateID, version.ID); err != nil {
			return nil, err
		}

		template.PublishedVersionID = &version.ID
		if err := s.repo.UpdateTemplate(ctx, template); err != nil {
			return nil, err
		}
		if err := s.repo.UpdateVersion(ctx, version); err != nil {
			return nil, err
		}
		if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
			TenantID:   cmd.TenantID,
			TemplateID: cmd.TemplateID,
			VersionID:  &version.ID,
			ActorID:    cmd.ActorUserID,
			Action:     domain.AuditPublished,
			Details:    map[string]any{},
			OccurredAt: s.clock.Now(),
		}); err != nil {
			return nil, err
		}
		return version, nil
	}

	if err := version.CanTransition(domain.VersionStatusDraft, hasReviewer); err != nil {
		return nil, err
	}
	version.Status = domain.VersionStatusDraft
	version.SubmittedAt = nil
	version.ReviewedAt = nil
	version.ApprovedAt = nil

	if err := s.repo.UpdateVersion(ctx, version); err != nil {
		return nil, err
	}
	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditRejected,
		Details: map[string]any{
			"reason": cmd.Reason,
			"stage":  "approver",
		},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return version, nil
}

type ArchiveCmd struct {
	TenantID, ActorUserID, TemplateID string
}

func (s *Service) ArchiveTemplate(ctx context.Context, cmd ArchiveCmd) (*domain.Template, error) {
	template, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID)
	if err != nil {
		return nil, err
	}
	if template.IsArchived() {
		return template, nil
	}

	now := s.clock.Now()
	template.ArchivedAt = &now

	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, err
	}
	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  nil,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditArchived,
		Details:    map[string]any{},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return template, nil
}

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
