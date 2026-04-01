package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

type DocumentTypeBundle struct {
	Definition domain.DocumentTypeDefinition
}

func (s *Service) ListDocumentTypeDefinitions(ctx context.Context) ([]domain.DocumentTypeDefinition, error) {
	items, err := s.repo.ListDocumentTypeDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		items = domain.DefaultDocumentTypeDefinitions()
	}
	out := make([]domain.DocumentTypeDefinition, 0, len(items))
	for _, item := range items {
		normalized, err := normalizeDocumentTypeDefinition(item)
		if err != nil {
			return nil, err
		}
		out = append(out, cloneDocumentTypeDefinition(normalized))
	}
	return out, nil
}

func (s *Service) GetDocumentTypeDefinition(ctx context.Context, key string) (domain.DocumentTypeDefinition, error) {
	normalizedKey := strings.ToLower(strings.TrimSpace(key))
	if normalizedKey == "" {
		return domain.DocumentTypeDefinition{}, domain.ErrInvalidCommand
	}

	item, err := s.repo.GetDocumentTypeDefinition(ctx, normalizedKey)
	if err == nil {
		normalized, normalizeErr := normalizeDocumentTypeDefinition(item)
		if normalizeErr != nil {
			return domain.DocumentTypeDefinition{}, normalizeErr
		}
		return cloneDocumentTypeDefinition(normalized), nil
	}

	for _, fallback := range domain.DefaultDocumentTypeDefinitions() {
		if strings.EqualFold(fallback.Key, normalizedKey) {
			normalized, normalizeErr := normalizeDocumentTypeDefinition(fallback)
			if normalizeErr != nil {
				return domain.DocumentTypeDefinition{}, normalizeErr
			}
			return cloneDocumentTypeDefinition(normalized), nil
		}
	}

	return domain.DocumentTypeDefinition{}, err
}

func (s *Service) UpsertDocumentTypeDefinition(ctx context.Context, item domain.DocumentTypeDefinition) error {
	normalized, err := normalizeDocumentTypeDefinition(item)
	if err != nil {
		return err
	}
	if err := validateDocumentTypeDefinitionSchema(normalized.Schema); err != nil {
		return err
	}
	return s.repo.UpsertDocumentTypeDefinition(ctx, normalized)
}

func (s *Service) GetDocumentTypeBundle(ctx context.Context, key string) (DocumentTypeBundle, error) {
	definition, err := s.GetDocumentTypeDefinition(ctx, key)
	if err != nil {
		return DocumentTypeBundle{}, err
	}
	return DocumentTypeBundle{Definition: definition}, nil
}

func (s *Service) resolveDocumentTypeDefinition(ctx context.Context, key string) (domain.DocumentTypeDefinition, error) {
	return s.GetDocumentTypeDefinition(ctx, key)
}
