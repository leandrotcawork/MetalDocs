package application

import (
	"context"
	"strings"

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

// GetCK5DocumentContentAuthorized returns the HTML body of the latest version
// for the given document. Mirrors the auth pattern of SaveBrowserContentAuthorized:
// GetDocument + isAllowed(CapabilityDocumentView). No MDDM template validation —
// CK5 manages its own template contract.
func (s *Service) GetCK5DocumentContentAuthorized(ctx context.Context, documentID string) (string, error) {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return "", err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentView)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", domain.ErrDocumentNotFound
	}
	ver, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return "", err
	}
	return ver.Content, nil
}

// SaveCK5DocumentContentAuthorized saves HTML content for a CK5 document.
// Auth pattern: GetDocument + isAllowed(CapabilityDocumentEdit). Uses CAS on
// the current content hash to prevent lost-update races.
func (s *Service) SaveCK5DocumentContentAuthorized(ctx context.Context, documentID, html string) error {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrDocumentNotFound
	}
	if !isVersioningAllowed(doc) {
		return domain.ErrVersioningNotAllowed
	}
	current, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return err
	}

	expectedHash := strings.TrimSpace(current.ContentHash)
	if expectedHash == "" {
		expectedHash = contentHash(current.Content)
	}

	updated := current
	updated.Content = html
	updated.ContentHash = contentHash(html)
	updated.ContentSource = domain.ContentSourceCK5Browser
	// CK5 content is HTML (not MDDM), so text extraction stays empty until a CK5-specific extractor is added.
	updated.TextContent = ""

	return s.repo.UpdateDraftVersionContentCAS(ctx, updated, expectedHash)
}
