package main

import (
	"context"
	"encoding/json"

	docapp "metaldocs/internal/modules/documents/application"
	pgrepo "metaldocs/internal/modules/documents/infrastructure/postgres"
	"metaldocs/internal/platform/render/docgen"
)

// mddmDocxRenderer adapts docgen.Client to the application.DocxRenderer interface.
// The release path sends the raw content-blocks envelope; metadata fields are
// left empty because the release repo is the authority for document metadata.
type mddmDocxRenderer struct {
	client *docgen.Client
}

func newMDDMDocxRenderer(client *docgen.Client) *mddmDocxRenderer {
	return &mddmDocxRenderer{client: client}
}

func (r *mddmDocxRenderer) RenderDocx(ctx context.Context, contentBlocks []byte) ([]byte, error) {
	payload := docgen.MDDMExportPayload{
		Envelope: json.RawMessage(contentBlocks),
		Metadata: docgen.MDDMExportMetadata{},
	}
	return r.client.GenerateMDDM(ctx, payload, "")
}

type mddmLoadRepoAdapter struct {
	repo *pgrepo.MDDMRepository
}

func (a *mddmLoadRepoAdapter) GetActiveDraft(ctx context.Context, documentID, userID string) (*docapp.LoadVersion, error) {
	if a == nil || a.repo == nil {
		return nil, nil
	}
	row, err := a.repo.GetActiveDraftForUser(ctx, documentID, userID)
	if err != nil || row == nil {
		return nil, err
	}
	return mapLoadVersion(row), nil
}

func (a *mddmLoadRepoAdapter) GetCurrentReleased(ctx context.Context, documentID string) (*docapp.LoadVersion, error) {
	if a == nil || a.repo == nil {
		return nil, nil
	}
	row, err := a.repo.GetCurrentReleased(ctx, documentID)
	if err != nil || row == nil {
		return nil, err
	}
	return mapLoadVersion(row), nil
}

func mapLoadVersion(row *pgrepo.DocumentVersion) *docapp.LoadVersion {
	if row == nil {
		return nil
	}
	return &docapp.LoadVersion{
		DocumentID:      row.DocumentID,
		Version:         row.VersionNumber,
		Status:          row.Status,
		Content:         json.RawMessage(row.ContentBlocks),
		TemplateKey:     readTemplateString(row.TemplateRef, "template_key", "template_id"),
		TemplateVersion: readTemplateInt(row.TemplateRef, "template_version"),
		ContentHash:     row.ContentHash,
	}
}

func readTemplateString(raw json.RawMessage, keys ...string) string {
	if len(raw) == 0 {
		return ""
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := parsed[key].(string); ok {
			return value
		}
	}
	return ""
}

func readTemplateInt(raw json.RawMessage, key string) int {
	if len(raw) == 0 {
		return 0
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return 0
	}
	switch value := parsed[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}
