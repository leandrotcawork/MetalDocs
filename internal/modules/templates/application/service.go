package application

import (
	"context"

	"metaldocs/internal/modules/templates/domain"
)

type Repository interface {
	CreateTemplate(ctx context.Context, t *domain.Template) (string, error)
	GetTemplate(ctx context.Context, id string) (*domain.Template, error)
	ListTemplates(ctx context.Context, tenantID string) ([]domain.TemplateListItem, error)
	CreateVersion(ctx context.Context, v *domain.TemplateVersion) (string, error)
	GetVersionByNum(ctx context.Context, templateID string, versionNum int) (*domain.TemplateVersion, error)
	UpdateDraftVersion(ctx context.Context, v *domain.TemplateVersion, expected int) error
	PublishVersion(ctx context.Context, versionID, by string) (newDraftID string, newVersionNum int, err error)
}

type DocgenValidator interface {
	ValidateTemplate(ctx context.Context, docxKey, schemaKey string) (valid bool, errs []byte, err error)
}

type Presigner interface {
	PresignTemplateDocxPUT(ctx context.Context, tenantID, templateID string, versionNum int) (url, storageKey string, err error)
	PresignTemplateSchemaPUT(ctx context.Context, tenantID, templateID string, versionNum int) (url, storageKey string, err error)
	PresignObjectGET(ctx context.Context, storageKey string) (url string, err error)
}

type Service struct {
	repo      Repository
	docgen    DocgenValidator
	presigner Presigner
}

func New(r Repository, d DocgenValidator, p Presigner) *Service {
	return &Service{repo: r, docgen: d, presigner: p}
}

func (s *Service) ListTemplates(ctx context.Context, tenantID string) ([]domain.TemplateListItem, error) {
	return s.repo.ListTemplates(ctx, tenantID)
}

func (s *Service) GetVersion(ctx context.Context, templateID string, versionNum int) (*domain.Template, *domain.TemplateVersion, error) {
	tpl, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, nil, err
	}
	ver, err := s.repo.GetVersionByNum(ctx, templateID, versionNum)
	if err != nil {
		return nil, nil, err
	}
	return tpl, ver, nil
}

func (s *Service) PresignDocxUpload(ctx context.Context, templateID string, versionNum int) (string, string, error) {
	tpl, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", err
	}
	return s.presigner.PresignTemplateDocxPUT(ctx, tpl.TenantID, templateID, versionNum)
}

func (s *Service) PresignSchemaUpload(ctx context.Context, templateID string, versionNum int) (string, string, error) {
	tpl, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", err
	}
	return s.presigner.PresignTemplateSchemaPUT(ctx, tpl.TenantID, templateID, versionNum)
}

func (s *Service) PresignObjectDownload(ctx context.Context, storageKey string) (string, error) {
	return s.presigner.PresignObjectGET(ctx, storageKey)
}

type CreateTemplateCmd struct {
	TenantID    string
	Key         string
	Name        string
	Description string
	CreatedBy   string
}

func (s *Service) CreateTemplate(ctx context.Context, cmd CreateTemplateCmd) (*domain.Template, *domain.TemplateVersion, error) {
	tpl := &domain.Template{
		TenantID:    cmd.TenantID,
		Key:         cmd.Key,
		Name:        cmd.Name,
		Description: cmd.Description,
		CreatedBy:   cmd.CreatedBy,
	}
	tplID, err := s.repo.CreateTemplate(ctx, tpl)
	if err != nil {
		return nil, nil, err
	}
	tpl.ID = tplID

	ver := domain.NewTemplateVersion(tplID, 1)
	ver.CreatedBy = cmd.CreatedBy
	verID, err := s.repo.CreateVersion(ctx, ver)
	if err != nil {
		return nil, nil, err
	}
	ver.ID = verID

	return tpl, ver, nil
}

type SaveDraftCmd struct {
	VersionID           string
	ExpectedLockVersion int
	DocxStorageKey      string
	SchemaStorageKey    string
	DocxContentHash     string
	SchemaContentHash   string
}

func (s *Service) SaveDraft(ctx context.Context, cmd SaveDraftCmd) error {
	ver := &domain.TemplateVersion{
		ID:                cmd.VersionID,
		DocxStorageKey:    cmd.DocxStorageKey,
		SchemaStorageKey:  cmd.SchemaStorageKey,
		DocxContentHash:   cmd.DocxContentHash,
		SchemaContentHash: cmd.SchemaContentHash,
	}
	return s.repo.UpdateDraftVersion(ctx, ver, cmd.ExpectedLockVersion)
}
