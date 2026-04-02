package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

func (s *Service) ResolveDocumentTemplate(ctx context.Context, documentID, profileCode string) (domain.DocumentTemplateVersion, error) {
	assignment, err := s.repo.GetDocumentTemplateAssignment(ctx, documentID)
	if err == nil {
		return s.repo.GetDocumentTemplateVersion(ctx, assignment.TemplateKey, assignment.TemplateVersion)
	}
	if !errors.Is(err, domain.ErrDocumentTemplateAssignmentNotFound) {
		return domain.DocumentTemplateVersion{}, err
	}
	return s.repo.GetDefaultDocumentTemplate(ctx, profileCode)
}

func (s *Service) resolveDocumentTemplateOptional(ctx context.Context, documentID, profileCode string) (domain.DocumentTemplateVersion, bool, error) {
	templateVersion, err := s.ResolveDocumentTemplate(ctx, documentID, profileCode)
	if err == nil {
		return templateVersion, true, nil
	}
	if errors.Is(err, domain.ErrDocumentTemplateNotFound) {
		return domain.DocumentTemplateVersion{}, false, nil
	}
	return domain.DocumentTemplateVersion{}, false, err
}

func draftTokenForVersion(version domain.Version) string {
	hash := strings.TrimSpace(version.ContentHash)
	if hash == "" {
		hash = contentHash(version.Content)
	}
	return fmt.Sprintf("v%d:%s", version.Number, hash)
}
