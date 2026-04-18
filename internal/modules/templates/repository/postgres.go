package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"metaldocs/internal/modules/templates/domain"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateTemplate(ctx context.Context, t *domain.Template) (string, error) {
	const q = `
INSERT INTO templates (tenant_id, key, name, description, created_by)
VALUES ($1,$2,$3,$4,$5) RETURNING id`
	var id string
	err := r.db.QueryRowContext(ctx, q, t.TenantID, t.Key, t.Name, t.Description, t.CreatedBy).Scan(&id)
	return id, err
}

func (r *Repository) GetTemplate(ctx context.Context, id string) (*domain.Template, error) {
	const q = `
SELECT id, tenant_id, key, name, coalesce(description,''), current_published_version_id,
       created_at, updated_at, created_by
FROM templates WHERE id = $1`
	t := &domain.Template{}
	var published sql.NullString
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&t.ID, &t.TenantID, &t.Key, &t.Name, &t.Description, &published,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	if published.Valid {
		t.CurrentPublishedVersionID = &published.String
	}
	return t, nil
}

func (r *Repository) ListTemplates(ctx context.Context, tenantID string) ([]domain.TemplateListItem, error) {
	const q = `
SELECT t.id, t.tenant_id, t.key, t.name, coalesce(t.description,''),
       t.created_at, t.updated_at, t.created_by,
       coalesce((SELECT MAX(version_num) FROM template_versions WHERE template_id=t.id), 1) AS latest_version,
       coalesce((SELECT id FROM template_versions WHERE template_id=t.id ORDER BY version_num DESC LIMIT 1), '00000000-0000-0000-0000-000000000000'::uuid)::text AS latest_version_id
FROM templates t
WHERE t.tenant_id=$1
ORDER BY t.updated_at DESC`
	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TemplateListItem{}
	for rows.Next() {
		var t domain.TemplateListItem
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Key, &t.Name, &t.Description,
			&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.LatestVersion, &t.LatestVersionID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetVersionByNum(ctx context.Context, templateID string, versionNum int) (*domain.TemplateVersion, error) {
	const q = `
SELECT id, template_id, version_num, status, grammar_version,
       coalesce(docx_storage_key,''), coalesce(schema_storage_key,''),
       coalesce(docx_content_hash,''), coalesce(schema_content_hash,''),
       lock_version, created_at, updated_at, created_by
FROM template_versions WHERE template_id=$1 AND version_num=$2`
	v := &domain.TemplateVersion{}
	var status string
	err := r.db.QueryRowContext(ctx, q, templateID, versionNum).Scan(
		&v.ID, &v.TemplateID, &v.VersionNum, &status, &v.GrammarVersion,
		&v.DocxStorageKey, &v.SchemaStorageKey,
		&v.DocxContentHash, &v.SchemaContentHash,
		&v.LockVersion, &v.CreatedAt, &v.UpdatedAt, &v.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	v.Status = domain.Status(status)
	return v, nil
}

func (r *Repository) CreateVersion(ctx context.Context, v *domain.TemplateVersion) (string, error) {
	const q = `
INSERT INTO template_versions
  (template_id, version_num, status, grammar_version,
   docx_storage_key, schema_storage_key,
   docx_content_hash, schema_content_hash,
   lock_version, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`
	var id string
	err := r.db.QueryRowContext(ctx, q,
		v.TemplateID, v.VersionNum, string(v.Status), v.GrammarVersion,
		v.DocxStorageKey, v.SchemaStorageKey,
		v.DocxContentHash, v.SchemaContentHash,
		v.LockVersion, v.CreatedBy,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", domain.ErrDuplicateDraft
		}
		return "", fmt.Errorf("insert version: %w", err)
	}
	return id, nil
}

func (r *Repository) UpdateDraftVersion(ctx context.Context, v *domain.TemplateVersion, expectedLock int) error {
	const q = `
UPDATE template_versions
SET docx_storage_key=$1, schema_storage_key=$2,
    docx_content_hash=$3, schema_content_hash=$4,
    lock_version = lock_version + 1,
    updated_at = now()
WHERE id=$5 AND status='draft' AND lock_version=$6`
	res, err := r.db.ExecContext(ctx, q,
		v.DocxStorageKey, v.SchemaStorageKey, v.DocxContentHash, v.SchemaContentHash,
		v.ID, expectedLock,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrLockVersionMismatch
	}
	return nil
}

func (r *Repository) PublishVersion(ctx context.Context, versionID, by string) (newDraftID string, newVersionNum int, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = tx.Rollback() }()

	const qUpdate = `
UPDATE template_versions
SET status='published', published_at=now(), published_by=$2
WHERE id=$1 AND status='draft'
RETURNING template_id, version_num, docx_storage_key, schema_storage_key, docx_content_hash, schema_content_hash`
	var (
		tplID, docxKey, schemaKey, docxHash, schemaHash string
		versionNum                                      int
	)
	if err := tx.QueryRowContext(ctx, qUpdate, versionID, by).
		Scan(&tplID, &versionNum, &docxKey, &schemaKey, &docxHash, &schemaHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, domain.ErrInvalidStateTransition
		}
		return "", 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE templates SET current_published_version_id=$1, updated_at=now() WHERE id=$2`,
		versionID, tplID); err != nil {
		return "", 0, err
	}
	newVersionNum = versionNum + 1
	const qInsert = `
INSERT INTO template_versions (
  template_id, version_num, status, docx_storage_key, schema_storage_key,
  docx_content_hash, schema_content_hash, lock_version, created_by
) VALUES ($1, $2, 'draft', $3, $4, $5, $6, 0, $7)
RETURNING id`
	if err := tx.QueryRowContext(ctx, qInsert,
		tplID, newVersionNum, docxKey, schemaKey, docxHash, schemaHash, by).Scan(&newDraftID); err != nil {
		return "", 0, err
	}
	return newDraftID, newVersionNum, tx.Commit()
}
