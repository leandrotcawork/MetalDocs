package application

import (
	"context"

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
