package application

import (
	"context"

	"metaldocs/internal/modules/documents/domain"
)

// GetCK5DocumentContent returns the HTML content and title of the latest
// ck5_browser version for the given document. Returns ErrDocumentNotFound
// if the document doesn't exist, is not accessible, or has no ck5_browser version.
func (s *Service) GetCK5DocumentContent(ctx context.Context, docID string) (html, title string, err error) {
	doc, err := s.GetDocumentAuthorized(ctx, docID)
	if err != nil {
		return "", "", domain.ErrDocumentNotFound
	}

	versions, err := s.repo.ListVersions(ctx, docID)
	if err != nil {
		return "", "", err
	}

	// Find the latest ck5_browser version (iterate in reverse; ListVersions returns sorted by number)
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].ContentSource == domain.ContentSourceCK5Browser {
			return versions[i].Content, doc.Title, nil
		}
	}

	return "", "", domain.ErrDocumentNotFound
}
