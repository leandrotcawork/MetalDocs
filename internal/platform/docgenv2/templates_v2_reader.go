package docgenv2

import (
	"context"
	"database/sql"
	"errors"
)

// TemplatesV2TemplateReader implements documents_v2/application.TemplateReader
// for templates authored via the templates_v2 module. Schema is stored in the
// database (not S3), so schemaKey is always "" and schemaJSON is always "".
type TemplatesV2TemplateReader struct {
	db *sql.DB
}

func NewTemplatesV2TemplateReader(db *sql.DB) *TemplatesV2TemplateReader {
	return &TemplatesV2TemplateReader{db: db}
}

func (r *TemplatesV2TemplateReader) GetPublishedVersion(ctx context.Context, tenantID, templateVersionID string) (docxKey, schemaKey, schemaJSON string, err error) {
	if r.db == nil {
		return "", "", "", errors.New("templates_v2 template reader: db is nil")
	}
	err = r.db.QueryRowContext(ctx, `
		SELECT tv.docx_storage_key
		FROM templates_v2_template_version tv
		JOIN templates_v2_templates tpl ON tpl.id = tv.template_id
		WHERE tv.id = $1
		  AND tpl.tenant_id = $2
		  AND tv.status = 'published'`,
		templateVersionID, tenantID,
	).Scan(&docxKey)
	if err != nil {
		return "", "", "", err
	}
	return docxKey, "", "", nil
}

// FanoutTemplateReader tries the primary reader first; if it returns sql.ErrNoRows,
// it falls back to the secondary reader.
type FanoutTemplateReader struct {
	primary   *TemplateReader
	secondary *TemplatesV2TemplateReader
}

func NewFanoutTemplateReader(primary *TemplateReader, secondary *TemplatesV2TemplateReader) *FanoutTemplateReader {
	return &FanoutTemplateReader{primary: primary, secondary: secondary}
}

func (f *FanoutTemplateReader) GetPublishedVersion(ctx context.Context, tenantID, templateVersionID string) (docxKey, schemaKey, schemaJSON string, err error) {
	docxKey, schemaKey, schemaJSON, err = f.primary.GetPublishedVersion(ctx, tenantID, templateVersionID)
	if err == nil {
		return docxKey, schemaKey, schemaJSON, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", "", err
	}
	return f.secondary.GetPublishedVersion(ctx, tenantID, templateVersionID)
}
