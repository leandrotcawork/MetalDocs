package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

// isInvalidUUID returns true when err is a Postgres error with SQLSTATE 22P02
// (invalid text representation). We map malformed UUID lookups to not found.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository { return &Repository{db: db} }

var _ application.Repository = (*Repository)(nil)

func (r *Repository) CreateTemplate(ctx context.Context, t *domain.Template) error {
	const q = `
INSERT INTO templates_v2_template (
	id, tenant_id, doc_type_code, key, name, description, areas, visibility,
	specific_areas, latest_version, published_version_id, created_by, created_at, archived_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8,
	$9, $10, $11, $12, $13, $14
)`
	_, err := r.db.ExecContext(ctx, q,
		t.ID, t.TenantID, t.DocTypeCode, t.Key, t.Name, t.Description, t.Areas, string(t.Visibility),
		t.SpecificAreas, t.LatestVersion, t.PublishedVersionID, t.CreatedBy, t.CreatedAt, t.ArchivedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrKeyConflict
		}
		return err
	}
	return nil
}

func (r *Repository) GetTemplate(ctx context.Context, tenantID, id string) (*domain.Template, error) {
	const q = `
SELECT
	id::text, tenant_id, doc_type_code, key, name, description, array_to_json(areas)::text, visibility, array_to_json(specific_areas)::text,
	latest_version, published_version_id::text, created_by, created_at, archived_at
FROM templates_v2_template
WHERE id = $1 AND tenant_id = $2`

	t, err := scanTemplate(r.db.QueryRowContext(ctx, q, id, tenantID))
	if errors.Is(err, sql.ErrNoRows) || isInvalidUUID(err) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) GetTemplateByKey(ctx context.Context, tenantID, key string) (*domain.Template, error) {
	const q = `
SELECT
	id::text, tenant_id, doc_type_code, key, name, description, array_to_json(areas)::text, visibility, array_to_json(specific_areas)::text,
	latest_version, published_version_id::text, created_by, created_at, archived_at
FROM templates_v2_template
WHERE tenant_id = $1 AND key = $2`

	t, err := scanTemplate(r.db.QueryRowContext(ctx, q, tenantID, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) ListTemplates(ctx context.Context, f application.ListFilter) ([]*domain.Template, error) {
	const q = `
SELECT
	id::text, tenant_id, doc_type_code, key, name, description, array_to_json(areas)::text, visibility, array_to_json(specific_areas)::text,
	latest_version, published_version_id::text, created_by, created_at, archived_at
FROM templates_v2_template
WHERE tenant_id = $1
  AND ($2::text IS NULL OR doc_type_code = $2)
  AND (cardinality($3::text[]) = 0 OR areas && $3::text[])
  AND (
    visibility = 'public'
    OR (visibility = 'internal' AND NOT $6::boolean)
    OR (visibility = 'specific' AND cardinality($7::text[]) > 0 AND specific_areas && $7::text[])
  )
ORDER BY created_at DESC
LIMIT $4 OFFSET $5`

	rows, err := r.db.QueryContext(
		ctx,
		q,
		f.TenantID,
		f.DocTypeCode,
		normalizedTextArray(f.AreaAny),
		f.Limit,
		f.Offset,
		f.IsExternalViewer,
		normalizedTextArray(f.ActorAreas),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.Template, 0)
	for rows.Next() {
		t, scanErr := scanTemplate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateTemplate(ctx context.Context, t *domain.Template) error {
	const q = `
UPDATE templates_v2_template
SET
	doc_type_code = $3,
	key = $4,
	name = $5,
	description = $6,
	areas = $7,
	visibility = $8,
	specific_areas = $9,
	latest_version = $10,
	published_version_id = $11,
	archived_at = $12
WHERE id = $1 AND tenant_id = $2`

	res, err := r.db.ExecContext(ctx, q,
		t.ID, t.TenantID, t.DocTypeCode, t.Key, t.Name, t.Description,
		t.Areas, string(t.Visibility), t.SpecificAreas, t.LatestVersion, t.PublishedVersionID, t.ArchivedAt,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateVersion(ctx context.Context, v *domain.TemplateVersion) error {
	metadataJSON, placeholderJSON, editableJSON, err := marshalVersionSchemas(v)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO templates_v2_template_version (
	id, template_id, version_number, status, docx_storage_key, content_hash,
	metadata_schema, placeholder_schema, editable_zones, author_id,
	pending_reviewer_role, pending_approver_role, reviewer_id, approver_id,
	submitted_at, reviewed_at, approved_at, published_at, obsoleted_at, created_at
) VALUES (
	$1, $2, $3, $4, $5, $6,
	$7, $8, $9, $10,
	$11, $12, $13, $14,
	$15, $16, $17, $18, $19, $20
)`
	_, err = r.db.ExecContext(ctx, q,
		v.ID, v.TemplateID, v.VersionNumber, string(v.Status), v.DocxStorageKey, v.ContentHash,
		metadataJSON, placeholderJSON, editableJSON, v.AuthorID,
		v.PendingReviewerRole, v.PendingApproverRole, v.ReviewerID, v.ApproverID,
		v.SubmittedAt, v.ReviewedAt, v.ApprovedAt, v.PublishedAt, v.ObsoletedAt, v.CreatedAt,
	)
	return err
}

func (r *Repository) GetVersion(ctx context.Context, templateID string, n int) (*domain.TemplateVersion, error) {
	const q = `
SELECT
	id::text, template_id::text, version_number, status, docx_storage_key, content_hash,
	metadata_schema, placeholder_schema, editable_zones, author_id,
	pending_reviewer_role, pending_approver_role, reviewer_id, approver_id,
	submitted_at, reviewed_at, approved_at, published_at, obsoleted_at, created_at
FROM templates_v2_template_version
WHERE template_id = $1 AND version_number = $2`

	v, err := scanTemplateVersion(r.db.QueryRowContext(ctx, q, templateID, n))
	if errors.Is(err, sql.ErrNoRows) || isInvalidUUID(err) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repository) GetVersionByID(ctx context.Context, id string) (*domain.TemplateVersion, error) {
	const q = `
SELECT
	id::text, template_id::text, version_number, status, docx_storage_key, content_hash,
	metadata_schema, placeholder_schema, editable_zones, author_id,
	pending_reviewer_role, pending_approver_role, reviewer_id, approver_id,
	submitted_at, reviewed_at, approved_at, published_at, obsoleted_at, created_at
FROM templates_v2_template_version
WHERE id = $1`

	v, err := scanTemplateVersion(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) || isInvalidUUID(err) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repository) UpdateVersion(ctx context.Context, v *domain.TemplateVersion) error {
	metadataJSON, placeholderJSON, editableJSON, err := marshalVersionSchemas(v)
	if err != nil {
		return err
	}

	const q = `
UPDATE templates_v2_template_version
SET
	status = $2,
	docx_storage_key = $3,
	content_hash = $4,
	metadata_schema = $5,
	placeholder_schema = $6,
	editable_zones = $7,
	pending_reviewer_role = $8,
	pending_approver_role = $9,
	reviewer_id = $10,
	approver_id = $11,
	submitted_at = $12,
	reviewed_at = $13,
	approved_at = $14,
	published_at = $15,
	obsoleted_at = $16
WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q,
		v.ID, string(v.Status), v.DocxStorageKey, v.ContentHash,
		metadataJSON, placeholderJSON, editableJSON,
		v.PendingReviewerRole, v.PendingApproverRole, v.ReviewerID, v.ApproverID,
		v.SubmittedAt, v.ReviewedAt, v.ApprovedAt, v.PublishedAt, v.ObsoletedAt,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ObsoletePreviousPublished(ctx context.Context, templateID, keepVersionID string) error {
	const q = `
UPDATE templates_v2_template_version
SET status = 'obsolete', obsoleted_at = now()
WHERE template_id = $1 AND status = 'published' AND id <> $2`
	_, err := r.db.ExecContext(ctx, q, templateID, keepVersionID)
	return err
}

func (r *Repository) GetApprovalConfig(ctx context.Context, templateID string) (*domain.ApprovalConfig, error) {
	const q = `
SELECT template_id::text, reviewer_role, approver_role
FROM templates_v2_approval_config
WHERE template_id = $1`
	var (
		cfg      domain.ApprovalConfig
		reviewer sql.NullString
	)
	err := r.db.QueryRowContext(ctx, q, templateID).Scan(&cfg.TemplateID, &reviewer, &cfg.ApproverRole)
	if errors.Is(err, sql.ErrNoRows) || isInvalidUUID(err) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if reviewer.Valid {
		cfg.ReviewerRole = &reviewer.String
	}
	return &cfg, nil
}

func (r *Repository) UpsertApprovalConfig(ctx context.Context, c *domain.ApprovalConfig) error {
	const q = `
INSERT INTO templates_v2_approval_config (template_id, reviewer_role, approver_role)
VALUES ($1, $2, $3)
ON CONFLICT (template_id) DO UPDATE
SET reviewer_role = EXCLUDED.reviewer_role,
    approver_role = EXCLUDED.approver_role`
	_, err := r.db.ExecContext(ctx, q, c.TemplateID, c.ReviewerRole, c.ApproverRole)
	return err
}

func (r *Repository) AppendAudit(ctx context.Context, e *domain.AuditEvent) error {
	detailsJSON, err := marshalAuditDetails(e.Details)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO templates_v2_audit_log (
	tenant_id, template_id, version_id, actor_id, action, details, occurred_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7
)`
	_, err = r.db.ExecContext(ctx, q,
		e.TenantID, e.TemplateID, e.VersionID, e.ActorID, string(e.Action), detailsJSON, e.OccurredAt,
	)
	return err
}

func (r *Repository) ListAudit(ctx context.Context, templateID string, limit, offset int) ([]*domain.AuditEvent, error) {
	const q = `
SELECT tenant_id, template_id::text, version_id::text, actor_id, action, details, occurred_at
FROM templates_v2_audit_log
WHERE template_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, q, templateID, limit, offset)
	if isInvalidUUID(err) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.AuditEvent, 0)
	for rows.Next() {
		var (
			event     domain.AuditEvent
			versionID sql.NullString
			details   []byte
		)
		if err := rows.Scan(&event.TenantID, &event.TemplateID, &versionID, &event.ActorID, &event.Action, &details, &event.OccurredAt); err != nil {
			return nil, err
		}
		if versionID.Valid {
			event.VersionID = &versionID.String
		}
		event.Details = map[string]any{}
		if len(details) > 0 {
			if err := unmarshalAuditDetails(details, &event.Details); err != nil {
				return nil, err
			}
		}
		out = append(out, &event)
	}
	return out, rows.Err()
}
