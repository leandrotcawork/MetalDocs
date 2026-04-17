package application

import (
	"context"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

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
