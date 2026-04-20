package application

import (
	"context"
	"time"

	"metaldocs/internal/modules/templates_v2/domain"
)

func (s *Service) GetTemplate(ctx context.Context, tenantID, id string) (*domain.Template, error) {
	t, err := s.repo.GetTemplate(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if t.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (s *Service) GetVersion(ctx context.Context, tenantID, templateID string, n int) (*domain.TemplateVersion, error) {
	if _, err := s.GetTemplate(ctx, tenantID, templateID); err != nil {
		return nil, err
	}
	return s.repo.GetVersion(ctx, templateID, n)
}

func (s *Service) ListTemplates(ctx context.Context, f ListFilter) ([]*domain.Template, error) {
	return s.repo.ListTemplates(ctx, f)
}

func (s *Service) ListAudit(ctx context.Context, tenantID, templateID string, limit, offset int) ([]*domain.AuditEvent, error) {
	if _, err := s.GetTemplate(ctx, tenantID, templateID); err != nil {
		return nil, err
	}
	return s.repo.ListAudit(ctx, templateID, limit, offset)
}

const docxDownloadTTL = 15 * time.Minute

type GetDocxURLCmd struct {
	TenantID, TemplateID string
	VersionNumber        int
}

func (s *Service) GetDocxURL(ctx context.Context, cmd GetDocxURLCmd) (string, error) {
	if _, err := s.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID); err != nil {
		return "", err
	}
	v, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return "", err
	}
	if v.DocxStorageKey == "" {
		return "", domain.ErrUploadMissing
	}
	return s.presign.PresignGET(ctx, v.DocxStorageKey, docxDownloadTTL)
}
