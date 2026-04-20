package application_test

import (
	"context"
	"fmt"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

type fakeRepo struct {
	templates       map[string]*domain.Template
	versions        map[string]*domain.TemplateVersion
	audit           []*domain.AuditEvent
	approvalConfigs map[string]*domain.ApprovalConfig
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		templates:       map[string]*domain.Template{},
		versions:        map[string]*domain.TemplateVersion{},
		audit:           []*domain.AuditEvent{},
		approvalConfigs: map[string]*domain.ApprovalConfig{},
	}
}

func (r *fakeRepo) CreateTemplate(_ context.Context, t *domain.Template) error {
	r.templates[t.ID] = t
	return nil
}

func (r *fakeRepo) GetTemplate(_ context.Context, tenantID, id string) (*domain.Template, error) {
	t, ok := r.templates[id]
	if !ok || t.TenantID != tenantID {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (r *fakeRepo) GetTemplateByKey(_ context.Context, tenantID, key string) (*domain.Template, error) {
	for _, t := range r.templates {
		if t.TenantID == tenantID && t.Key == key {
			return t, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) ListTemplates(_ context.Context, f application.ListFilter) ([]*domain.Template, error) {
	out := make([]*domain.Template, 0, len(r.templates))
	for _, t := range r.templates {
		if f.TenantID != "" && t.TenantID != f.TenantID {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, domain.ErrNotFound
	}
	return out, nil
}

func (r *fakeRepo) UpdateTemplate(_ context.Context, t *domain.Template) error {
	if _, ok := r.templates[t.ID]; !ok {
		return domain.ErrNotFound
	}
	r.templates[t.ID] = t
	return nil
}

func (r *fakeRepo) CreateVersion(_ context.Context, v *domain.TemplateVersion) error {
	r.versions[v.ID] = v
	return nil
}

func (r *fakeRepo) GetVersion(_ context.Context, templateID string, n int) (*domain.TemplateVersion, error) {
	for _, v := range r.versions {
		if v.TemplateID == templateID && v.VersionNumber == n {
			return v, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepo) GetVersionByID(_ context.Context, id string) (*domain.TemplateVersion, error) {
	v, ok := r.versions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return v, nil
}

func (r *fakeRepo) UpdateVersion(_ context.Context, v *domain.TemplateVersion) error {
	if _, ok := r.versions[v.ID]; !ok {
		return domain.ErrNotFound
	}
	r.versions[v.ID] = v
	return nil
}

func (r *fakeRepo) ObsoletePreviousPublished(_ context.Context, templateID, keepVersionID string) error {
	found := false
	for _, v := range r.versions {
		if v.TemplateID != templateID {
			continue
		}
		found = true
		if v.Status == domain.VersionStatusPublished && v.ID != keepVersionID {
			now := time.Now().UTC()
			v.ObsoletedAt = &now
		}
	}
	if !found {
		return domain.ErrNotFound
	}
	return nil
}

func (r *fakeRepo) GetApprovalConfig(_ context.Context, templateID string) (*domain.ApprovalConfig, error) {
	c, ok := r.approvalConfigs[templateID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return c, nil
}

func (r *fakeRepo) UpsertApprovalConfig(_ context.Context, c *domain.ApprovalConfig) error {
	r.approvalConfigs[c.TemplateID] = c
	return nil
}

func (r *fakeRepo) AppendAudit(_ context.Context, e *domain.AuditEvent) error {
	r.audit = append(r.audit, e)
	return nil
}

func (r *fakeRepo) ListAudit(_ context.Context, templateID string, limit, offset int) ([]*domain.AuditEvent, error) {
	matched := make([]*domain.AuditEvent, 0, len(r.audit))
	for _, e := range r.audit {
		if e.TemplateID == templateID {
			matched = append(matched, e)
		}
	}
	if len(matched) == 0 {
		return nil, domain.ErrNotFound
	}
	if offset >= len(matched) {
		return nil, domain.ErrNotFound
	}
	end := len(matched)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return matched[offset:end], nil
}

type fakePresigner struct {
	HeadResult   string
	HeadErr      error
	DeleteCalled int
}

func (p *fakePresigner) PresignPUT(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://presigned/" + key, nil
}

func (p *fakePresigner) HeadContentHash(_ context.Context, _ string) (string, error) {
	if p.HeadResult == "" {
		return "hash_abc", p.HeadErr
	}
	return p.HeadResult, p.HeadErr
}

func (p *fakePresigner) Delete(_ context.Context, _ string) error {
	p.DeleteCalled++
	return nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time {
	return time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
}

type fakeUUID struct {
	counter int
}

func (u *fakeUUID) New() string {
	u.counter++
	return fmt.Sprintf("id_%d", u.counter)
}
