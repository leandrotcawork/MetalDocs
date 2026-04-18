package docgenv2

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

type TemplateReader struct {
	db     *sql.DB
	client *minio.Client
	bucket string
}

func NewTemplateReader(db *sql.DB, client *minio.Client, bucket string) *TemplateReader {
	return &TemplateReader{db: db, client: client, bucket: bucket}
}

func (t *TemplateReader) GetPublishedVersion(ctx context.Context, tenantID, templateVersionID string) (docxKey, schemaKey, schemaJSON string, err error) {
	if t.db == nil {
		return "", "", "", errors.New("template reader db is nil")
	}
	if err := t.db.QueryRowContext(ctx, `
		SELECT tv.docx_storage_key, tv.schema_storage_key
		FROM template_versions tv
		JOIN templates tpl ON tpl.id = tv.template_id
		WHERE tv.id = $1
		  AND tpl.tenant_id = $2
		  AND tv.status = 'published'`,
		templateVersionID, tenantID,
	).Scan(&docxKey, &schemaKey); err != nil {
		return "", "", "", err
	}

	if t.client == nil {
		return docxKey, schemaKey, "", errors.New("template reader minio client is nil")
	}
	obj, err := t.client.GetObject(ctx, t.bucket, schemaKey, minio.GetObjectOptions{})
	if err != nil {
		return "", "", "", err
	}
	defer obj.Close()
	if _, err := obj.Stat(); err != nil {
		return "", "", "", err
	}
	payload, err := io.ReadAll(obj)
	if err != nil {
		return "", "", "", fmt.Errorf("read schema object: %w", err)
	}
	return docxKey, schemaKey, string(payload), nil
}
