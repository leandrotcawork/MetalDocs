package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

func (s *Service) ListVersions(ctx context.Context, documentID string) ([]domain.Version, error) {
	if strings.TrimSpace(documentID) == "" {
		return nil, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return nil, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, domain.ErrDocumentNotFound
	}
	return s.repo.ListVersions(ctx, strings.TrimSpace(documentID))
}

func (s *Service) GetDocumentAuthorized(ctx context.Context, documentID string) (domain.Document, error) {
	if strings.TrimSpace(documentID) == "" {
		return domain.Document{}, domain.ErrInvalidCommand
	}
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.Document{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return domain.Document{}, err
	}
	if !allowed {
		return domain.Document{}, domain.ErrDocumentNotFound
	}
	return doc, nil
}
