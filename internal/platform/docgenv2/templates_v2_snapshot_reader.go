package docgenv2

import (
	"context"
	"database/sql"
	"errors"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
)

var _ application.SnapshotTemplateReader = (*TemplatesV2SnapshotReader)(nil)

type TemplatesV2SnapshotReader struct {
	db *sql.DB
}

func NewTemplatesV2SnapshotReader(db *sql.DB) *TemplatesV2SnapshotReader {
	return &TemplatesV2SnapshotReader{db: db}
}

func (r *TemplatesV2SnapshotReader) LoadForSnapshot(ctx context.Context, tenantID, templateVersionID string) (domain.TemplateSnapshot, error) {
	if r.db == nil {
		return domain.TemplateSnapshot{}, errors.New("templates_v2 snapshot reader: db is nil")
	}

	var phJSON, docxKey string
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(tv.placeholder_schema::text, '{}'),
		       COALESCE(tv.docx_storage_key, '')
		  FROM templates_v2_template_version tv
		  JOIN templates_v2_template tpl ON tpl.id = tv.template_id
		 WHERE tv.id = $1::uuid
		   AND tpl.tenant_id = $2::uuid
		   AND tv.status = 'published'`,
		templateVersionID, tenantID,
	).Scan(&phJSON, &docxKey)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.TemplateSnapshot{}, domain.ErrSnapshotTemplateNotFound
	}
	if err != nil {
		return domain.TemplateSnapshot{}, err
	}

	return domain.TemplateSnapshot{
		PlaceholderSchemaJSON: []byte(phJSON),
		CompositionJSON:       []byte("{}"),
		BodyDocxBytes:         nil,
		BodyDocxS3Key:         docxKey,
	}, nil
}
