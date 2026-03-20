package application

import (
	"context"

	"metaldocs/internal/modules/documents/domain"
)

func (s *Service) latestVersion(ctx context.Context, documentID string) (domain.Version, error) {
	versions, err := s.repo.ListVersions(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}
	if len(versions) == 0 {
		return domain.Version{}, domain.ErrVersionNotFound
	}
	return versions[len(versions)-1], nil
}

func isVersioningAllowed(doc domain.Document) bool {
	return doc.Status == domain.StatusDraft || doc.Status == domain.StatusInReview
}
