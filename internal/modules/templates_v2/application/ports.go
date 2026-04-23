package application

import (
	"context"
	"time"

	"metaldocs/internal/modules/templates_v2/domain"
)

type Repository interface {
	CreateTemplate(ctx context.Context, t *domain.Template) error
	GetTemplate(ctx context.Context, tenantID, id string) (*domain.Template, error)
	GetTemplateByKey(ctx context.Context, tenantID, key string) (*domain.Template, error)
	ListTemplates(ctx context.Context, f ListFilter) ([]*domain.Template, error)
	UpdateTemplate(ctx context.Context, t *domain.Template) error

	CreateVersion(ctx context.Context, v *domain.TemplateVersion) error
	GetVersion(ctx context.Context, templateID string, n int) (*domain.TemplateVersion, error)
	GetVersionByID(ctx context.Context, id string) (*domain.TemplateVersion, error)
	UpdateVersion(ctx context.Context, v *domain.TemplateVersion) error
	ObsoletePreviousPublished(ctx context.Context, templateID, keepVersionID string) error

	GetApprovalConfig(ctx context.Context, templateID string) (*domain.ApprovalConfig, error)
	UpsertApprovalConfig(ctx context.Context, c *domain.ApprovalConfig) error

	AppendAudit(ctx context.Context, e *domain.AuditEvent) error
	ListAudit(ctx context.Context, templateID string, limit, offset int) ([]*domain.AuditEvent, error)
}

type Presigner interface {
	PresignPUT(ctx context.Context, key string, expires time.Duration) (url string, err error)
	PresignGET(ctx context.Context, key string, expires time.Duration) (url string, err error)
	HeadContentHash(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type Clock interface{ Now() time.Time }
type UUIDGen interface{ New() string }
type ResolverRegistryReader interface{ Known() map[string]int }

type ListFilter struct {
	TenantID         string
	AreaAny          []string
	ActorAreas       []string
	IsExternalViewer bool
	DocTypeCode      *string
	Status           *domain.VersionStatus
	Limit            int
	Offset           int
}
