package application

import (
	"context"
	"fmt"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

type DocumentRuntimeBundle struct {
	Document domain.Document
	Version  domain.Version
	Type     domain.DocumentTypeDefinition
}

func (s *Service) GetDocumentRuntimeBundle(ctx context.Context, documentID string) (DocumentRuntimeBundle, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	version, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	typeDefinition, err := s.resolveDocumentTypeDefinition(ctx, firstNonEmpty(doc.DocumentType, doc.DocumentProfile))
	if err != nil {
		return DocumentRuntimeBundle{}, err
	}

	return DocumentRuntimeBundle{
		Document: doc,
		Version:  version,
		Type:     typeDefinition,
	}, nil
}

func (s *Service) SaveDocumentValues(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	documentID := strings.TrimSpace(cmd.DocumentID)
	if documentID == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	typeDefinition, err := s.resolveDocumentTypeDefinition(ctx, firstNonEmpty(doc.DocumentType, doc.DocumentProfile))
	if err != nil {
		return domain.Version{}, err
	}

	values := cloneRuntimeValues(cmd.Values)
	if err := validateDocumentTypeValues(typeDefinition.Schema, values); err != nil {
		return domain.Version{}, err
	}

	latest, err := s.latestVersion(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	if doc.Status == domain.StatusDraft {
		updated := latest
		updated.Values = values
		if err := s.repo.UpdateVersionValues(ctx, documentID, updated.Number, values); err != nil {
			return domain.Version{}, err
		}
		return updated, nil
	}

	next, err := s.repo.NextVersionNumber(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	nextVersion := latest
	nextVersion.Number = next
	nextVersion.Values = values
	nextVersion.CreatedAt = s.clock.Now()
	nextVersion.ChangeSummary = fmt.Sprintf("Runtime values update %d", next)

	if err := s.repo.SaveVersion(ctx, nextVersion); err != nil {
		return domain.Version{}, err
	}

	return nextVersion, nil
}

func (s *Service) SaveDocumentValuesAuthorized(ctx context.Context, cmd domain.SaveDocumentValuesCommand) (domain.Version, error) {
	documentID := strings.TrimSpace(cmd.DocumentID)
	if documentID == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.Version{}, err
	}
	if !allowed {
		return domain.Version{}, domain.ErrDocumentNotFound
	}

	return s.SaveDocumentValues(ctx, cmd)
}
