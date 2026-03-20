package application

import (
	"context"
	"errors"

	"metaldocs/internal/modules/documents/domain"
)

type DocumentEditorBundle struct {
	Document   domain.Document
	Versions   []domain.Version
	Schema     domain.DocumentProfileSchemaVersion
	Governance domain.DocumentProfileGovernance
	Presence   []domain.CollaborationPresence
	EditLock   *domain.DocumentEditLock
}

func (s *Service) GetDocumentEditorBundle(ctx context.Context, documentID string) (DocumentEditorBundle, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return DocumentEditorBundle{}, err
	}

	versions, err := s.repo.ListVersions(ctx, doc.ID)
	if err != nil {
		return DocumentEditorBundle{}, err
	}

	schema, err := s.resolveActiveProfileSchema(ctx, doc.DocumentProfile)
	if err != nil {
		return DocumentEditorBundle{}, err
	}

	governance, err := s.GetDocumentProfileGovernance(ctx, doc.DocumentProfile)
	if err != nil {
		return DocumentEditorBundle{}, err
	}

	presence, err := s.ListCollaborationPresenceAuthorized(ctx, doc.ID)
	if err != nil {
		return DocumentEditorBundle{}, err
	}

	var editLock *domain.DocumentEditLock
	lock, err := s.GetDocumentEditLockAuthorized(ctx, doc.ID)
	if err != nil {
		if !errors.Is(err, domain.ErrEditLockNotFound) {
			return DocumentEditorBundle{}, err
		}
	} else {
		editLock = &lock
	}

	return DocumentEditorBundle{
		Document:   doc,
		Versions:   versions,
		Schema:     schema,
		Governance: governance,
		Presence:   presence,
		EditLock:   editLock,
	}, nil
}
