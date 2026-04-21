package application

import (
	"context"
	"encoding/json"
	"time"

	"metaldocs/internal/modules/taxonomy/domain"
)

type TemplateVersionChecker interface {
	IsPublished(ctx context.Context, versionID string) (bool, string, error)
}

type ProfileService struct {
	profiles  domain.ProfileRepository
	tplCheck  TemplateVersionChecker
	govLogger domain.GovernanceLogger
	now       func() time.Time
}

func NewProfileService(
	profiles domain.ProfileRepository,
	tplCheck TemplateVersionChecker,
	govLogger domain.GovernanceLogger,
) *ProfileService {
	return &ProfileService{
		profiles:  profiles,
		tplCheck:  tplCheck,
		govLogger: govLogger,
		now:       time.Now,
	}
}

func (s *ProfileService) SetDefaultTemplate(ctx context.Context, tenantID, profileCode, templateVersionID, actorID string) error {
	profile, err := s.profiles.GetByCode(ctx, tenantID, profileCode)
	if err != nil {
		return err
	}
	if !profile.IsActive() {
		return domain.ErrProfileArchived
	}

	published, templateProfileCode, err := s.tplCheck.IsPublished(ctx, templateVersionID)
	if err != nil {
		return err
	}
	if !published {
		return domain.ErrTemplateNotPublished
	}
	if templateProfileCode != profileCode {
		return domain.ErrTemplateProfileMismatch
	}

	profile.DefaultTemplateVersionID = &templateVersionID
	if err := s.profiles.Update(ctx, profile); err != nil {
		return err
	}

	if s.govLogger == nil {
		return nil
	}

	payload, _ := json.Marshal(map[string]string{
		"template_version_id": templateVersionID,
	})
	return s.govLogger.Log(ctx, domain.GovernanceEvent{
		TenantID:     tenantID,
		EventType:    "profile.default_template_change",
		ActorUserID:  actorID,
		ResourceType: "document_profile",
		ResourceID:   profileCode,
		PayloadJSON:  payload,
	})
}

func (s *ProfileService) Archive(ctx context.Context, tenantID, profileCode, _ string) error {
	profile, err := s.profiles.GetByCode(ctx, tenantID, profileCode)
	if err != nil {
		return err
	}
	if err := profile.Archive(s.now()); err != nil {
		return err
	}
	return s.profiles.Update(ctx, profile)
}
