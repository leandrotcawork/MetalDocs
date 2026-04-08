package application

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain"
)

type ExportRepo interface {
	GetVersion(ctx context.Context, versionID uuid.UUID) (*exportVersion, error)
}

type exportVersion struct {
	Status        string
	DocxBytes     []byte
	ContentBlocks []byte
}

type ExportService struct {
	repo     ExportRepo
	renderer DocxRenderer
}

func NewExportService(repo ExportRepo, renderer DocxRenderer) *ExportService {
	return &ExportService{repo: repo, renderer: renderer}
}

func (s *ExportService) ExportDocx(ctx context.Context, versionID uuid.UUID, mode string) ([]byte, error) {
	version, err := s.repo.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, domain.ErrVersionNotFound
	}

	switch strings.ToLower(strings.TrimSpace(version.Status)) {
	case "released", "archived":
		if len(version.DocxBytes) == 0 {
			return nil, domain.ErrInvalidCommand
		}
		return version.DocxBytes, nil
	case "draft", "pending_approval":
		if !isValidExportMode(mode) {
			return nil, domain.ErrInvalidCommand
		}
		if s.renderer == nil {
			return nil, domain.ErrRenderUnavailable
		}
		return s.renderer.RenderDocx(ctx, version.ContentBlocks)
	default:
		return nil, domain.ErrInvalidCommand
	}
}

func isValidExportMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "debug", "production":
		return true
	default:
		return false
	}
}
