package application

import (
	"context"
	"fmt"
	"html"
	"strings"

	"metaldocs/internal/modules/documents/domain"
)

type BrowserEditorBundle struct {
	Document         domain.Document
	Versions         []domain.Version
	Governance       domain.DocumentProfileGovernance
	TemplateSnapshot domain.DocumentTemplateSnapshot
	Body             string
	DraftToken       string
}

func (s *Service) GetBrowserEditorBundleAuthorized(ctx context.Context, documentID string) (BrowserEditorBundle, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return BrowserEditorBundle{}, err
	}

	versions, err := s.repo.ListVersions(ctx, doc.ID)
	if err != nil {
		return BrowserEditorBundle{}, err
	}
	if len(versions) == 0 {
		return BrowserEditorBundle{}, domain.ErrVersionNotFound
	}

	governance, err := s.GetDocumentProfileGovernance(ctx, doc.DocumentProfile)
	if err != nil {
		return BrowserEditorBundle{}, err
	}

	current := versions[len(versions)-1]
	templateVersion, hasTemplate, err := s.resolveBrowserTemplateVersionForVersion(ctx, doc, current)
	if err != nil {
		return BrowserEditorBundle{}, err
	}
	if !hasTemplate {
		return BrowserEditorBundle{}, domain.ErrDocumentTemplateNotFound
	}

	bundle := BrowserEditorBundle{
		Document:   doc,
		Versions:   versions,
		Governance: governance,
		Body:       current.Content,
		DraftToken: draftTokenForVersion(current),
	}
	bundle.TemplateSnapshot = documentTemplateSnapshotFromVersion(templateVersion)
	return bundle, nil
}

func (s *Service) SaveBrowserContentAuthorized(ctx context.Context, cmd domain.SaveBrowserContentCommand) (domain.Version, error) {
	if strings.TrimSpace(cmd.DocumentID) == "" || strings.TrimSpace(cmd.DraftToken) == "" {
		return domain.Version{}, domain.ErrInvalidCommand
	}

	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(cmd.DocumentID))
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
	if !isVersioningAllowed(doc) {
		return domain.Version{}, domain.ErrVersioningNotAllowed
	}

	current, err := s.latestVersion(ctx, doc.ID)
	if err != nil {
		return domain.Version{}, err
	}
	if !matchesDraftToken(cmd.DraftToken, current) {
		return domain.Version{}, domain.ErrDraftConflict
	}

	templateVersion, hasTemplate, err := s.resolveBrowserTemplateVersionForVersion(ctx, doc, current)
	if err != nil {
		return domain.Version{}, err
	}
	if !hasTemplate {
		return domain.Version{}, domain.ErrDocumentTemplateNotFound
	}

	version := current
	version.Content = cmd.Body
	version.ContentHash = contentHash(cmd.Body)
	version.ChangeSummary = fmt.Sprintf("Content version %d", current.Number)
	version.ContentSource = domain.ContentSourceBrowserEditor
	version.TextContent = plainTextFromMDDM(cmd.Body)
	version.TemplateKey = templateVersion.TemplateKey
	version.TemplateVersion = templateVersion.Version

	expectedHash := strings.TrimSpace(current.ContentHash)
	if expectedHash == "" {
		expectedHash = contentHash(current.Content)
	}
	if err := s.repo.UpdateDraftVersionContentCAS(ctx, version, expectedHash); err != nil {
		return domain.Version{}, err
	}
	return version, nil
}

func (s *Service) resolveTemplateVersionForVersion(ctx context.Context, doc domain.Document, version domain.Version) (domain.DocumentTemplateVersion, bool, error) {
	if strings.TrimSpace(version.TemplateKey) != "" && version.TemplateVersion > 0 {
		item, err := s.repo.GetDocumentTemplateVersion(ctx, version.TemplateKey, version.TemplateVersion)
		if err != nil {
			return domain.DocumentTemplateVersion{}, false, err
		}
		if err := s.validateDocumentTemplateCompatibility(ctx, item); err != nil {
			return domain.DocumentTemplateVersion{}, false, err
		}
		return item, true, nil
	}
	return domain.DocumentTemplateVersion{}, false, domain.ErrDocumentTemplateNotFound
}

func (s *Service) resolveBrowserTemplateVersionForVersion(ctx context.Context, doc domain.Document, version domain.Version) (domain.DocumentTemplateVersion, bool, error) {
	item, hasTemplate, err := s.resolveTemplateVersionForVersion(ctx, doc, version)
	if err != nil {
		return domain.DocumentTemplateVersion{}, false, err
	}
	if !hasTemplate {
		return domain.DocumentTemplateVersion{}, false, nil
	}
	if err := validateBrowserTemplateVersion(item); err != nil {
		return domain.DocumentTemplateVersion{}, false, domain.ErrInvalidCommand
	}
	return item, true, nil
}

func validateBrowserTemplateVersion(item domain.DocumentTemplateVersion) error {
	if strings.TrimSpace(item.TemplateKey) == "" || item.Version <= 0 || strings.TrimSpace(item.ProfileCode) == "" {
		return domain.ErrInvalidCommand
	}
	if !item.IsBrowserEditor() {
		return domain.ErrInvalidCommand
	}
	return nil
}

// substituteTemplateTokens replaces well-known placeholder tokens in the body
// with real document metadata. Called when serving the browser editor bundle so
// the user sees pre-populated fields without having to type them.
func substituteTemplateTokens(body string, doc domain.Document, version domain.Version) string {
	versao := fmt.Sprintf("%02d", version.Number)
	data := "—"
	if !doc.CreatedAt.IsZero() {
		data = doc.CreatedAt.Format("02/01/2006")
	}
	por := html.EscapeString(doc.OwnerID)
	if por == "" {
		por = "—"
	}
	body = strings.ReplaceAll(body, "{{versao}}", versao)
	body = strings.ReplaceAll(body, "{{data_criacao}}", data)
	body = strings.ReplaceAll(body, "{{elaborador}}", por)
	return body
}

func documentTemplateSnapshotFromVersion(item domain.DocumentTemplateVersion) domain.DocumentTemplateSnapshot {
	return domain.DocumentTemplateSnapshot{
		TemplateKey:   item.TemplateKey,
		Version:       item.Version,
		ProfileCode:   item.ProfileCode,
		SchemaVersion: item.SchemaVersion,
		Editor:        item.Editor,
		ContentFormat: item.ContentFormat,
		Body:          item.Body,
		Definition:    item.Definition,
		ExportConfig:  item.ExportConfig,
	}
}
