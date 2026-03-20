package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateDocument(ctx context.Context, document domain.Document) error {
	const q = `
INSERT INTO metaldocs.documents (
  id, title, document_type_code, document_profile_code, document_family_code, process_area_code, subject_code,
  profile_schema_version, owner_id, business_unit, department, classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, $16, $17::jsonb, $18, $19)
`
	tagsJSON, metadataJSON, effectiveAt, expiryAt := serializeDocument(document)
	_, err := r.db.ExecContext(ctx, q,
		document.ID,
		document.Title,
		document.DocumentType,
		document.DocumentProfile,
		document.DocumentFamily,
		nullIfEmpty(document.ProcessArea),
		nullIfEmpty(document.Subject),
		document.ProfileSchemaVersion,
		document.OwnerID,
		document.BusinessUnit,
		document.Department,
		document.Classification,
		document.Status,
		tagsJSON,
		effectiveAt,
		expiryAt,
		metadataJSON,
		document.CreatedAt,
		document.UpdatedAt,
	)
	if err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) CreateDocumentWithInitialVersion(ctx context.Context, document domain.Document, version domain.Version) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx create document: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const insertDoc = `
INSERT INTO metaldocs.documents (
  id, title, document_type_code, document_profile_code, document_family_code, process_area_code, subject_code,
  profile_schema_version, owner_id, business_unit, department, classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, $16, $17::jsonb, $18, $19)
`
	tagsJSON, metadataJSON, effectiveAt, expiryAt := serializeDocument(document)
	if _, err := tx.ExecContext(ctx, insertDoc,
		document.ID,
		document.Title,
		document.DocumentType,
		document.DocumentProfile,
		document.DocumentFamily,
		nullIfEmpty(document.ProcessArea),
		nullIfEmpty(document.Subject),
		document.ProfileSchemaVersion,
		document.OwnerID,
		document.BusinessUnit,
		document.Department,
		document.Classification,
		document.Status,
		tagsJSON,
		effectiveAt,
		expiryAt,
		metadataJSON,
		document.CreatedAt,
		document.UpdatedAt,
	); err != nil {
		return mapError(err)
	}

	const insertVersion = `
INSERT INTO metaldocs.document_versions (
  document_id, version_number, content, content_hash, change_summary,
  content_source, native_content, docx_storage_key, pdf_storage_key, text_content,
  file_size_bytes, original_filename, page_count, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14)
`
	contentSource, nativeContentJSON, textContent := serializeVersion(version)
	if _, err := tx.ExecContext(ctx, insertVersion,
		version.DocumentID,
		version.Number,
		version.Content,
		version.ContentHash,
		version.ChangeSummary,
		contentSource,
		nativeContentJSON,
		nullIfEmpty(version.DocxStorageKey),
		nullIfEmpty(version.PdfStorageKey),
		textContent,
		nullIfZeroInt64(version.FileSizeBytes),
		nullIfEmpty(version.OriginalFilename),
		nullIfZeroInt(version.PageCount),
		version.CreatedAt,
	); err != nil {
		return mapError(err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx create document: %w", err)
	}
	return nil
}

func (r *Repository) CreateDocumentWithInitialVersionAndPolicies(ctx context.Context, document domain.Document, version domain.Version, policies []domain.AccessPolicy) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx create document with policies: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const insertDoc = `
INSERT INTO metaldocs.documents (
  id, title, document_type_code, document_profile_code, document_family_code, process_area_code, subject_code,
  profile_schema_version, owner_id, business_unit, department, classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, $16, $17::jsonb, $18, $19)
`
	tagsJSON, metadataJSON, effectiveAt, expiryAt := serializeDocument(document)
	if _, err := tx.ExecContext(ctx, insertDoc,
		document.ID,
		document.Title,
		document.DocumentType,
		document.DocumentProfile,
		document.DocumentFamily,
		nullIfEmpty(document.ProcessArea),
		nullIfEmpty(document.Subject),
		document.ProfileSchemaVersion,
		document.OwnerID,
		document.BusinessUnit,
		document.Department,
		document.Classification,
		document.Status,
		tagsJSON,
		effectiveAt,
		expiryAt,
		metadataJSON,
		document.CreatedAt,
		document.UpdatedAt,
	); err != nil {
		return mapError(err)
	}

	const insertVersion = `
INSERT INTO metaldocs.document_versions (
  document_id, version_number, content, content_hash, change_summary,
  content_source, native_content, docx_storage_key, pdf_storage_key, text_content,
  file_size_bytes, original_filename, page_count, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14)
`
	contentSource, nativeContentJSON, textContent := serializeVersion(version)
	if _, err := tx.ExecContext(ctx, insertVersion,
		version.DocumentID,
		version.Number,
		version.Content,
		version.ContentHash,
		version.ChangeSummary,
		contentSource,
		nativeContentJSON,
		nullIfEmpty(version.DocxStorageKey),
		nullIfEmpty(version.PdfStorageKey),
		textContent,
		nullIfZeroInt64(version.FileSizeBytes),
		nullIfEmpty(version.OriginalFilename),
		nullIfZeroInt(version.PageCount),
		version.CreatedAt,
	); err != nil {
		return mapError(err)
	}

	if len(policies) > 0 {
		const insertPolicy = `
INSERT INTO metaldocs.document_access_policies (
  subject_type, subject_id, resource_scope, resource_id, capability, effect, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
`
		for _, policy := range policies {
			if _, err := tx.ExecContext(ctx, insertPolicy,
				policy.SubjectType,
				policy.SubjectID,
				policy.ResourceScope,
				policy.ResourceID,
				policy.Capability,
				policy.Effect,
			); err != nil {
				return mapError(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx create document with policies: %w", err)
	}
	return nil
}

func (r *Repository) GetDocument(ctx context.Context, documentID string) (domain.Document, error) {
	const q = `
SELECT id, title, document_type_code, document_profile_code, document_family_code, process_area_code, subject_code, profile_schema_version,
       owner_id, business_unit, department, classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
FROM metaldocs.documents
WHERE id = $1
`
	var doc domain.Document
	var tagsJSON []byte
	var metadataJSON []byte
	var effectiveAt sql.NullTime
	var expiryAt sql.NullTime
	var processArea sql.NullString
	var subject sql.NullString
	err := r.db.QueryRowContext(ctx, q, documentID).Scan(
		&doc.ID,
		&doc.Title,
		&doc.DocumentType,
		&doc.DocumentProfile,
		&doc.DocumentFamily,
		&processArea,
		&subject,
		&doc.ProfileSchemaVersion,
		&doc.OwnerID,
		&doc.BusinessUnit,
		&doc.Department,
		&doc.Classification,
		&doc.Status,
		&tagsJSON,
		&effectiveAt,
		&expiryAt,
		&metadataJSON,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Document{}, domain.ErrDocumentNotFound
		}
		return domain.Document{}, fmt.Errorf("get document: %w", err)
	}
	doc.ProcessArea = strings.TrimSpace(processArea.String)
	doc.Subject = strings.TrimSpace(subject.String)
	applyOptionalFields(&doc, tagsJSON, metadataJSON, effectiveAt, expiryAt)
	return doc, nil
}

func (r *Repository) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	const q = `
SELECT id, title, document_type_code, document_profile_code, document_family_code, process_area_code, subject_code, profile_schema_version,
       owner_id, business_unit, department, classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
FROM metaldocs.documents
ORDER BY created_at ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var out []domain.Document
	for rows.Next() {
		var doc domain.Document
		var tagsJSON []byte
		var metadataJSON []byte
		var effectiveAt sql.NullTime
		var expiryAt sql.NullTime
		var processArea sql.NullString
		var subject sql.NullString
		if err := rows.Scan(
			&doc.ID,
			&doc.Title,
			&doc.DocumentType,
			&doc.DocumentProfile,
			&doc.DocumentFamily,
			&processArea,
			&subject,
			&doc.ProfileSchemaVersion,
			&doc.OwnerID,
			&doc.BusinessUnit,
			&doc.Department,
			&doc.Classification,
			&doc.Status,
			&tagsJSON,
			&effectiveAt,
			&expiryAt,
			&metadataJSON,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		doc.ProcessArea = strings.TrimSpace(processArea.String)
		doc.Subject = strings.TrimSpace(subject.String)
		applyOptionalFields(&doc, tagsJSON, metadataJSON, effectiveAt, expiryAt)
		out = append(out, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents rows: %w", err)
	}

	return out, nil
}

func (r *Repository) ListDocumentsForReviewReminder(ctx context.Context, fromInclusive, toInclusive time.Time) ([]domain.Document, error) {
	const q = `
SELECT id, title, document_type_code, document_profile_code, document_family_code, process_area_code, subject_code, profile_schema_version,
       owner_id, business_unit, department, classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
FROM metaldocs.documents
WHERE expiry_at IS NOT NULL
  AND expiry_at >= $1
  AND expiry_at <= $2
  AND status IN ('APPROVED', 'PUBLISHED')
ORDER BY expiry_at ASC, created_at ASC
`
	rows, err := r.db.QueryContext(ctx, q, fromInclusive.UTC(), toInclusive.UTC())
	if err != nil {
		return nil, fmt.Errorf("list documents for review reminder: %w", err)
	}
	defer rows.Close()

	var out []domain.Document
	for rows.Next() {
		var doc domain.Document
		var tagsJSON []byte
		var metadataJSON []byte
		var effectiveAt sql.NullTime
		var expiryAt sql.NullTime
		var processArea sql.NullString
		var subject sql.NullString
		if err := rows.Scan(
			&doc.ID,
			&doc.Title,
			&doc.DocumentType,
			&doc.DocumentProfile,
			&doc.DocumentFamily,
			&processArea,
			&subject,
			&doc.ProfileSchemaVersion,
			&doc.OwnerID,
			&doc.BusinessUnit,
			&doc.Department,
			&doc.Classification,
			&doc.Status,
			&tagsJSON,
			&effectiveAt,
			&expiryAt,
			&metadataJSON,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan document for review reminder: %w", err)
		}
		doc.ProcessArea = strings.TrimSpace(processArea.String)
		doc.Subject = strings.TrimSpace(subject.String)
		applyOptionalFields(&doc, tagsJSON, metadataJSON, effectiveAt, expiryAt)
		out = append(out, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents for review reminder rows: %w", err)
	}

	return out, nil
}

func (r *Repository) ListDocumentTypes(ctx context.Context) ([]domain.DocumentType, error) {
	const q = `
SELECT p.code, p.name, p.description, COALESCE(g.review_interval_days, p.review_interval_days)
FROM metaldocs.document_profiles p
LEFT JOIN metaldocs.document_profile_governance g ON g.profile_code = p.code
WHERE p.is_active = TRUE
ORDER BY p.code ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list document types: %w", err)
	}
	defer rows.Close()

	var out []domain.DocumentType
	for rows.Next() {
		var item domain.DocumentType
		if err := rows.Scan(&item.Code, &item.Name, &item.Description, &item.ReviewIntervalDays); err != nil {
			return nil, fmt.Errorf("scan document type: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document types rows: %w", err)
	}
	return out, nil
}

func (r *Repository) ListDocumentFamilies(ctx context.Context) ([]domain.DocumentFamily, error) {
	const q = `
SELECT code, name, description
FROM metaldocs.document_families
WHERE is_active = TRUE
ORDER BY code ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list document families: %w", err)
	}
	defer rows.Close()

	var out []domain.DocumentFamily
	for rows.Next() {
		var item domain.DocumentFamily
		if err := rows.Scan(&item.Code, &item.Name, &item.Description); err != nil {
			return nil, fmt.Errorf("scan document family: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document families rows: %w", err)
	}
	return out, nil
}

func (r *Repository) ListDocumentProfiles(ctx context.Context) ([]domain.DocumentProfile, error) {
	const q = `
SELECT p.code, p.family_code, p.name, p.alias, p.description,
       COALESCE(g.review_interval_days, p.review_interval_days) AS review_interval_days,
       COALESCE(s.active_version, 1) AS active_schema_version,
       COALESCE(g.workflow_profile, 'standard_approval') AS workflow_profile,
       COALESCE(g.approval_required, TRUE) AS approval_required,
       COALESCE(g.retention_days, 0) AS retention_days,
       COALESCE(g.validity_days, 0) AS validity_days
FROM metaldocs.document_profiles p
LEFT JOIN (
  SELECT profile_code, MAX(version) FILTER (WHERE is_active) AS active_version
  FROM metaldocs.document_profile_schema_versions
  GROUP BY profile_code
) s ON s.profile_code = p.code
LEFT JOIN metaldocs.document_profile_governance g ON g.profile_code = p.code
WHERE p.is_active = TRUE
ORDER BY code ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list document profiles: %w", err)
	}
	defer rows.Close()

	var out []domain.DocumentProfile
	for rows.Next() {
		var item domain.DocumentProfile
		if err := rows.Scan(
			&item.Code,
			&item.FamilyCode,
			&item.Name,
			&item.Alias,
			&item.Description,
			&item.ReviewIntervalDays,
			&item.ActiveSchemaVersion,
			&item.WorkflowProfile,
			&item.ApprovalRequired,
			&item.RetentionDays,
			&item.ValidityDays,
		); err != nil {
			return nil, fmt.Errorf("scan document profile: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document profiles rows: %w", err)
	}
	return out, nil
}

func (r *Repository) UpsertDocumentProfile(ctx context.Context, item domain.DocumentProfile) error {
	const q = `
INSERT INTO metaldocs.document_profiles (code, family_code, name, alias, description, review_interval_days, is_active)
VALUES ($1, $2, $3, $4, $5, $6, TRUE)
ON CONFLICT (code) DO UPDATE
SET family_code = EXCLUDED.family_code,
    name = EXCLUDED.name,
    alias = EXCLUDED.alias,
    description = EXCLUDED.description,
    review_interval_days = EXCLUDED.review_interval_days,
    is_active = TRUE
`
	if _, err := r.db.ExecContext(ctx, q,
		item.Code,
		item.FamilyCode,
		item.Name,
		item.Alias,
		item.Description,
		item.ReviewIntervalDays,
	); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) DeactivateDocumentProfile(ctx context.Context, code string) error {
	const q = `
UPDATE metaldocs.document_profiles
SET is_active = FALSE
WHERE code = $1 AND is_active = TRUE
`
	result, err := r.db.ExecContext(ctx, q, code)
	if err != nil {
		return mapError(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrInvalidCommand
	}
	return nil
}

func (r *Repository) ListDocumentProfileSchemas(ctx context.Context, profileCode string) ([]domain.DocumentProfileSchemaVersion, error) {
	const q = `
SELECT profile_code, version, is_active, metadata_rules_json
FROM metaldocs.document_profile_schema_versions
WHERE ($1 = '' OR profile_code = $1)
ORDER BY profile_code ASC, version ASC
`
	rows, err := r.db.QueryContext(ctx, q, profileCode)
	if err != nil {
		return nil, fmt.Errorf("list document profile schemas: %w", err)
	}
	defer rows.Close()

	var out []domain.DocumentProfileSchemaVersion
	for rows.Next() {
		var item domain.DocumentProfileSchemaVersion
		var rawRules []byte
		if err := rows.Scan(&item.ProfileCode, &item.Version, &item.IsActive, &rawRules); err != nil {
			return nil, fmt.Errorf("scan document profile schema: %w", err)
		}
		if len(rawRules) > 0 {
			if err := json.Unmarshal(rawRules, &item.MetadataRules); err != nil {
				return nil, fmt.Errorf("unmarshal document profile schema rules: %w", err)
			}
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document profile schemas rows: %w", err)
	}
	return out, nil
}

func (r *Repository) UpsertDocumentProfileSchemaVersion(ctx context.Context, item domain.DocumentProfileSchemaVersion) error {
	rawRules, err := json.Marshal(item.MetadataRules)
	if err != nil {
		return fmt.Errorf("marshal document profile schema rules: %w", err)
	}

	if item.IsActive {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx upsert document profile schema version: %w", err)
		}
		defer func() {
			_ = tx.Rollback()
		}()

		if _, err := tx.ExecContext(ctx,
			`UPDATE metaldocs.document_profile_schema_versions SET is_active = FALSE WHERE profile_code = $1`,
			item.ProfileCode,
		); err != nil {
			return fmt.Errorf("deactivate schema versions: %w", err)
		}

		const q = `
INSERT INTO metaldocs.document_profile_schema_versions (profile_code, version, metadata_rules_json, is_active)
VALUES ($1, $2, $3::jsonb, TRUE)
ON CONFLICT (profile_code, version) DO UPDATE
SET metadata_rules_json = EXCLUDED.metadata_rules_json,
    is_active = TRUE
`
		if _, err := tx.ExecContext(ctx, q, item.ProfileCode, item.Version, rawRules); err != nil {
			return mapError(err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit tx upsert document profile schema version: %w", err)
		}
		return nil
	}

	const q = `
INSERT INTO metaldocs.document_profile_schema_versions (profile_code, version, metadata_rules_json, is_active)
VALUES ($1, $2, $3::jsonb, FALSE)
ON CONFLICT (profile_code, version) DO UPDATE
SET metadata_rules_json = EXCLUDED.metadata_rules_json
`
	if _, err := r.db.ExecContext(ctx, q, item.ProfileCode, item.Version, rawRules); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) ActivateDocumentProfileSchemaVersion(ctx context.Context, profileCode string, version int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx activate schema version: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx,
		`UPDATE metaldocs.document_profile_schema_versions SET is_active = FALSE WHERE profile_code = $1`,
		profileCode,
	); err != nil {
		return fmt.Errorf("deactivate schema versions: %w", err)
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE metaldocs.document_profile_schema_versions SET is_active = TRUE WHERE profile_code = $1 AND version = $2`,
		profileCode,
		version,
	)
	if err != nil {
		return mapError(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrInvalidCommand
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx activate schema version: %w", err)
	}
	return nil
}

func (r *Repository) GetDocumentProfileGovernance(ctx context.Context, profileCode string) (domain.DocumentProfileGovernance, error) {
	const q = `
SELECT profile_code, workflow_profile, review_interval_days, approval_required, retention_days, validity_days
FROM metaldocs.document_profile_governance
WHERE profile_code = $1
`
	var item domain.DocumentProfileGovernance
	if err := r.db.QueryRowContext(ctx, q, profileCode).Scan(
		&item.ProfileCode,
		&item.WorkflowProfile,
		&item.ReviewIntervalDays,
		&item.ApprovalRequired,
		&item.RetentionDays,
		&item.ValidityDays,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.DocumentProfileGovernance{}, domain.ErrInvalidCommand
		}
		return domain.DocumentProfileGovernance{}, fmt.Errorf("get document profile governance: %w", err)
	}
	return item, nil
}

func (r *Repository) UpsertDocumentProfileGovernance(ctx context.Context, item domain.DocumentProfileGovernance) error {
	const q = `
INSERT INTO metaldocs.document_profile_governance (
  profile_code, workflow_profile, review_interval_days, approval_required, retention_days, validity_days
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (profile_code) DO UPDATE
SET workflow_profile = EXCLUDED.workflow_profile,
    review_interval_days = EXCLUDED.review_interval_days,
    approval_required = EXCLUDED.approval_required,
    retention_days = EXCLUDED.retention_days,
    validity_days = EXCLUDED.validity_days
`
	if _, err := r.db.ExecContext(ctx, q,
		item.ProfileCode,
		item.WorkflowProfile,
		item.ReviewIntervalDays,
		item.ApprovalRequired,
		item.RetentionDays,
		item.ValidityDays,
	); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) ListProcessAreas(ctx context.Context) ([]domain.ProcessArea, error) {
	const q = `
SELECT code, name, description
FROM metaldocs.document_process_areas
WHERE is_active = TRUE
ORDER BY code ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list process areas: %w", err)
	}
	defer rows.Close()

	var out []domain.ProcessArea
	for rows.Next() {
		var item domain.ProcessArea
		if err := rows.Scan(&item.Code, &item.Name, &item.Description); err != nil {
			return nil, fmt.Errorf("scan process area: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list process areas rows: %w", err)
	}
	return out, nil
}

func (r *Repository) ListDocumentDepartments(ctx context.Context) ([]domain.DocumentDepartment, error) {
	const q = `
SELECT code, name, description
FROM metaldocs.document_departments
WHERE is_active = TRUE
ORDER BY code ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list document departments: %w", err)
	}
	defer rows.Close()

	var out []domain.DocumentDepartment
	for rows.Next() {
		var item domain.DocumentDepartment
		if err := rows.Scan(&item.Code, &item.Name, &item.Description); err != nil {
			return nil, fmt.Errorf("scan document department: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document departments rows: %w", err)
	}
	return out, nil
}

func (r *Repository) UpsertProcessArea(ctx context.Context, item domain.ProcessArea) error {
	const q = `
INSERT INTO metaldocs.document_process_areas (code, name, description, is_active)
VALUES ($1, $2, $3, TRUE)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_active = TRUE
`
	if _, err := r.db.ExecContext(ctx, q, item.Code, item.Name, item.Description); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) UpsertDocumentDepartment(ctx context.Context, item domain.DocumentDepartment) error {
	const q = `
INSERT INTO metaldocs.document_departments (code, name, description, is_active)
VALUES ($1, $2, $3, TRUE)
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_active = TRUE
`
	if _, err := r.db.ExecContext(ctx, q, item.Code, item.Name, item.Description); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) DeactivateProcessArea(ctx context.Context, code string) error {
	const q = `
UPDATE metaldocs.document_process_areas
SET is_active = FALSE
WHERE code = $1 AND is_active = TRUE
`
	result, err := r.db.ExecContext(ctx, q, code)
	if err != nil {
		return mapError(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrInvalidCommand
	}
	return nil
}

func (r *Repository) DeactivateDocumentDepartment(ctx context.Context, code string) error {
	const q = `
UPDATE metaldocs.document_departments
SET is_active = FALSE
WHERE code = $1 AND is_active = TRUE
`
	result, err := r.db.ExecContext(ctx, q, code)
	if err != nil {
		return mapError(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrInvalidCommand
	}
	return nil
}

func (r *Repository) ListSubjects(ctx context.Context) ([]domain.Subject, error) {
	const q = `
SELECT code, process_area_code, name, description
FROM metaldocs.document_subjects
WHERE is_active = TRUE
ORDER BY process_area_code ASC, code ASC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	var out []domain.Subject
	for rows.Next() {
		var item domain.Subject
		if err := rows.Scan(&item.Code, &item.ProcessAreaCode, &item.Name, &item.Description); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list subjects rows: %w", err)
	}
	return out, nil
}

func (r *Repository) UpsertSubject(ctx context.Context, item domain.Subject) error {
	const q = `
INSERT INTO metaldocs.document_subjects (code, process_area_code, name, description, is_active)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (code) DO UPDATE
SET process_area_code = EXCLUDED.process_area_code,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    is_active = TRUE
`
	if _, err := r.db.ExecContext(ctx, q, item.Code, item.ProcessAreaCode, item.Name, item.Description); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) DeactivateSubject(ctx context.Context, code string) error {
	const q = `
UPDATE metaldocs.document_subjects
SET is_active = FALSE
WHERE code = $1 AND is_active = TRUE
`
	result, err := r.db.ExecContext(ctx, q, code)
	if err != nil {
		return mapError(err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrInvalidCommand
	}
	return nil
}

func (r *Repository) ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]domain.AccessPolicy, error) {
	const q = `
SELECT subject_type, subject_id, resource_scope, resource_id, capability, effect
FROM metaldocs.document_access_policies
WHERE resource_scope = $1 AND resource_id = $2
ORDER BY subject_type ASC, subject_id ASC, capability ASC
`
	rows, err := r.db.QueryContext(ctx, q, resourceScope, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list access policies: %w", err)
	}
	defer rows.Close()

	var out []domain.AccessPolicy
	for rows.Next() {
		var item domain.AccessPolicy
		if err := rows.Scan(
			&item.SubjectType,
			&item.SubjectID,
			&item.ResourceScope,
			&item.ResourceID,
			&item.Capability,
			&item.Effect,
		); err != nil {
			return nil, fmt.Errorf("scan access policy: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list access policies rows: %w", err)
	}
	return out, nil
}

func (r *Repository) ReplaceAccessPolicies(ctx context.Context, resourceScope, resourceID string, policies []domain.AccessPolicy) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx replace access policies: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM metaldocs.document_access_policies WHERE resource_scope = $1 AND resource_id = $2`,
		resourceScope,
		resourceID,
	); err != nil {
		return fmt.Errorf("delete access policies: %w", err)
	}

	if len(policies) > 0 {
		const insertPolicy = `
INSERT INTO metaldocs.document_access_policies (
  subject_type, subject_id, resource_scope, resource_id, capability, effect, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
`
		for _, policy := range policies {
			if _, err := tx.ExecContext(ctx, insertPolicy,
				policy.SubjectType,
				policy.SubjectID,
				policy.ResourceScope,
				policy.ResourceID,
				policy.Capability,
				policy.Effect,
			); err != nil {
				return mapError(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx replace access policies: %w", err)
	}
	return nil
}

func (r *Repository) UpdateDocumentStatus(ctx context.Context, documentID, status string) error {
	const q = `
UPDATE metaldocs.documents
SET status = $2, updated_at = NOW()
WHERE id = $1
`
	res, err := r.db.ExecContext(ctx, q, documentID, status)
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected update document status: %w", err)
	}
	if affected == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *Repository) SaveVersion(ctx context.Context, version domain.Version) error {
	const q = `
INSERT INTO metaldocs.document_versions (
  document_id, version_number, content, content_hash, change_summary,
  content_source, native_content, docx_storage_key, pdf_storage_key, text_content,
  file_size_bytes, original_filename, page_count, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14)
`
	contentSource, nativeContentJSON, textContent := serializeVersion(version)
	_, err := r.db.ExecContext(ctx, q,
		version.DocumentID,
		version.Number,
		version.Content,
		version.ContentHash,
		version.ChangeSummary,
		contentSource,
		nativeContentJSON,
		nullIfEmpty(version.DocxStorageKey),
		nullIfEmpty(version.PdfStorageKey),
		textContent,
		nullIfZeroInt64(version.FileSizeBytes),
		nullIfEmpty(version.OriginalFilename),
		nullIfZeroInt(version.PageCount),
		version.CreatedAt,
	)
	if err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) ListVersions(ctx context.Context, documentID string) ([]domain.Version, error) {
	_, err := r.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	const q = `
SELECT document_id, version_number, content, content_hash, change_summary,
       content_source, native_content, docx_storage_key, pdf_storage_key, text_content,
       file_size_bytes, original_filename, page_count, created_at
FROM metaldocs.document_versions
WHERE document_id = $1
ORDER BY version_number ASC
`
	rows, err := r.db.QueryContext(ctx, q, documentID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var out []domain.Version
	for rows.Next() {
		var version domain.Version
		var nativeContentJSON []byte
		var docxStorageKey sql.NullString
		var pdfStorageKey sql.NullString
		var textContent sql.NullString
		var fileSizeBytes sql.NullInt64
		var originalFilename sql.NullString
		var pageCount sql.NullInt64
		if err := rows.Scan(
			&version.DocumentID,
			&version.Number,
			&version.Content,
			&version.ContentHash,
			&version.ChangeSummary,
			&version.ContentSource,
			&nativeContentJSON,
			&docxStorageKey,
			&pdfStorageKey,
			&textContent,
			&fileSizeBytes,
			&originalFilename,
			&pageCount,
			&version.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		applyVersionOptionalFields(&version, nativeContentJSON, docxStorageKey, pdfStorageKey, textContent, fileSizeBytes, originalFilename, pageCount)
		out = append(out, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list versions rows: %w", err)
	}

	return out, nil
}

func (r *Repository) GetVersion(ctx context.Context, documentID string, versionNumber int) (domain.Version, error) {
	_, err := r.GetDocument(ctx, documentID)
	if err != nil {
		return domain.Version{}, err
	}

	const q = `
SELECT document_id, version_number, content, content_hash, change_summary,
       content_source, native_content, docx_storage_key, pdf_storage_key, text_content,
       file_size_bytes, original_filename, page_count, created_at
FROM metaldocs.document_versions
WHERE document_id = $1 AND version_number = $2
`
	var version domain.Version
	var nativeContentJSON []byte
	var docxStorageKey sql.NullString
	var pdfStorageKey sql.NullString
	var textContent sql.NullString
	var fileSizeBytes sql.NullInt64
	var originalFilename sql.NullString
	var pageCount sql.NullInt64
	if err := r.db.QueryRowContext(ctx, q, documentID, versionNumber).Scan(
		&version.DocumentID,
		&version.Number,
		&version.Content,
		&version.ContentHash,
		&version.ChangeSummary,
		&version.ContentSource,
		&nativeContentJSON,
		&docxStorageKey,
		&pdfStorageKey,
		&textContent,
		&fileSizeBytes,
		&originalFilename,
		&pageCount,
		&version.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.Version{}, domain.ErrVersionNotFound
		}
		return domain.Version{}, fmt.Errorf("get version: %w", err)
	}
	applyVersionOptionalFields(&version, nativeContentJSON, docxStorageKey, pdfStorageKey, textContent, fileSizeBytes, originalFilename, pageCount)
	return version, nil
}

func (r *Repository) NextVersionNumber(ctx context.Context, documentID string) (int, error) {
	_, err := r.GetDocument(ctx, documentID)
	if err != nil {
		return 0, err
	}

	const q = `
SELECT COALESCE(MAX(version_number), 0) + 1
FROM metaldocs.document_versions
WHERE document_id = $1
`
	var next int
	if err := r.db.QueryRowContext(ctx, q, documentID).Scan(&next); err != nil {
		return 0, fmt.Errorf("next version number: %w", err)
	}
	return next, nil
}

func (r *Repository) CreateAttachment(ctx context.Context, attachment domain.Attachment) error {
	const q = `
INSERT INTO metaldocs.document_attachments (
  id, document_id, file_name, content_type, size_bytes, storage_key, uploaded_by, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	_, err := r.db.ExecContext(ctx, q,
		attachment.ID,
		attachment.DocumentID,
		attachment.FileName,
		attachment.ContentType,
		attachment.SizeBytes,
		attachment.StorageKey,
		attachment.UploadedBy,
		attachment.CreatedAt,
	)
	if err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) GetAttachment(ctx context.Context, attachmentID string) (domain.Attachment, error) {
	const q = `
SELECT id, document_id, file_name, content_type, size_bytes, storage_key, uploaded_by, created_at
FROM metaldocs.document_attachments
WHERE id = $1
`
	var attachment domain.Attachment
	if err := r.db.QueryRowContext(ctx, q, attachmentID).Scan(
		&attachment.ID,
		&attachment.DocumentID,
		&attachment.FileName,
		&attachment.ContentType,
		&attachment.SizeBytes,
		&attachment.StorageKey,
		&attachment.UploadedBy,
		&attachment.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.Attachment{}, domain.ErrAttachmentNotFound
		}
		return domain.Attachment{}, fmt.Errorf("get attachment: %w", err)
	}
	return attachment, nil
}

func (r *Repository) ListAttachments(ctx context.Context, documentID string) ([]domain.Attachment, error) {
	_, err := r.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	const q = `
SELECT id, document_id, file_name, content_type, size_bytes, storage_key, uploaded_by, created_at
FROM metaldocs.document_attachments
WHERE document_id = $1
ORDER BY created_at ASC
`
	rows, err := r.db.QueryContext(ctx, q, documentID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()

	var out []domain.Attachment
	for rows.Next() {
		var attachment domain.Attachment
		if err := rows.Scan(
			&attachment.ID,
			&attachment.DocumentID,
			&attachment.FileName,
			&attachment.ContentType,
			&attachment.SizeBytes,
			&attachment.StorageKey,
			&attachment.UploadedBy,
			&attachment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		out = append(out, attachment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list attachments rows: %w", err)
	}
	return out, nil
}

func (r *Repository) UpsertCollaborationPresence(ctx context.Context, item domain.CollaborationPresence) error {
	const q = `
INSERT INTO metaldocs.document_collaboration_presence (
  document_id, user_id, display_name, last_seen_at, created_at, updated_at
)
VALUES ($1, $2, $3, $4, NOW(), NOW())
ON CONFLICT (document_id, user_id) DO UPDATE
SET display_name = EXCLUDED.display_name,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = NOW()
`
	if _, err := r.db.ExecContext(ctx, q, item.DocumentID, item.UserID, item.DisplayName, item.LastSeenAt.UTC()); err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) ListCollaborationPresence(ctx context.Context, documentID string, activeSince time.Time) ([]domain.CollaborationPresence, error) {
	const q = `
SELECT document_id, user_id, display_name, last_seen_at
FROM metaldocs.document_collaboration_presence
WHERE document_id = $1
  AND last_seen_at >= $2
ORDER BY last_seen_at DESC
`
	rows, err := r.db.QueryContext(ctx, q, documentID, activeSince.UTC())
	if err != nil {
		return nil, fmt.Errorf("list collaboration presence: %w", err)
	}
	defer rows.Close()

	items := make([]domain.CollaborationPresence, 0)
	for rows.Next() {
		var item domain.CollaborationPresence
		if err := rows.Scan(&item.DocumentID, &item.UserID, &item.DisplayName, &item.LastSeenAt); err != nil {
			return nil, fmt.Errorf("scan collaboration presence: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list collaboration presence rows: %w", err)
	}
	return items, nil
}

func (r *Repository) AcquireDocumentEditLock(ctx context.Context, item domain.DocumentEditLock, now time.Time) (domain.DocumentEditLock, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.DocumentEditLock{}, fmt.Errorf("begin tx acquire edit lock: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const lockQuery = `
SELECT document_id, locked_by, display_name, lock_reason, acquired_at, expires_at
FROM metaldocs.document_edit_locks
WHERE document_id = $1
FOR UPDATE
`
	var current domain.DocumentEditLock
	lockErr := tx.QueryRowContext(ctx, lockQuery, item.DocumentID).Scan(
		&current.DocumentID,
		&current.LockedBy,
		&current.DisplayName,
		&current.LockReason,
		&current.AcquiredAt,
		&current.ExpiresAt,
	)
	if lockErr != nil && lockErr != sql.ErrNoRows {
		return domain.DocumentEditLock{}, fmt.Errorf("query current edit lock: %w", lockErr)
	}
	if lockErr == nil && current.ExpiresAt.After(now.UTC()) && !strings.EqualFold(current.LockedBy, item.LockedBy) {
		return domain.DocumentEditLock{}, domain.ErrEditLockActive
	}

	const upsert = `
INSERT INTO metaldocs.document_edit_locks (
  document_id, locked_by, display_name, lock_reason, acquired_at, expires_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (document_id) DO UPDATE
SET locked_by = EXCLUDED.locked_by,
    display_name = EXCLUDED.display_name,
    lock_reason = EXCLUDED.lock_reason,
    acquired_at = EXCLUDED.acquired_at,
    expires_at = EXCLUDED.expires_at,
    updated_at = NOW()
`
	if _, err := tx.ExecContext(ctx, upsert,
		item.DocumentID,
		item.LockedBy,
		item.DisplayName,
		item.LockReason,
		item.AcquiredAt.UTC(),
		item.ExpiresAt.UTC(),
	); err != nil {
		return domain.DocumentEditLock{}, mapError(err)
	}
	if err := tx.Commit(); err != nil {
		return domain.DocumentEditLock{}, fmt.Errorf("commit acquire edit lock: %w", err)
	}
	return item, nil
}

func (r *Repository) GetDocumentEditLock(ctx context.Context, documentID string, now time.Time) (domain.DocumentEditLock, error) {
	const q = `
SELECT document_id, locked_by, display_name, lock_reason, acquired_at, expires_at
FROM metaldocs.document_edit_locks
WHERE document_id = $1
`
	var item domain.DocumentEditLock
	if err := r.db.QueryRowContext(ctx, q, documentID).Scan(
		&item.DocumentID,
		&item.LockedBy,
		&item.DisplayName,
		&item.LockReason,
		&item.AcquiredAt,
		&item.ExpiresAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.DocumentEditLock{}, domain.ErrEditLockNotFound
		}
		return domain.DocumentEditLock{}, fmt.Errorf("get document edit lock: %w", err)
	}
	if !item.ExpiresAt.After(now.UTC()) {
		return domain.DocumentEditLock{}, domain.ErrEditLockNotFound
	}
	return item, nil
}

func (r *Repository) ReleaseDocumentEditLock(ctx context.Context, documentID, lockedBy string) error {
	const selectCurrent = `
SELECT locked_by, expires_at
FROM metaldocs.document_edit_locks
WHERE document_id = $1
`
	var currentLockedBy string
	var expiresAt time.Time
	if err := r.db.QueryRowContext(ctx, selectCurrent, documentID).Scan(&currentLockedBy, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.ErrEditLockNotFound
		}
		return fmt.Errorf("release document edit lock lookup: %w", err)
	}
	if expiresAt.After(time.Now().UTC()) && !strings.EqualFold(currentLockedBy, lockedBy) {
		return domain.ErrEditLockActive
	}

	const q = `
DELETE FROM metaldocs.document_edit_locks
WHERE document_id = $1
`
	result, err := r.db.ExecContext(ctx, q, documentID)
	if err != nil {
		return fmt.Errorf("release document edit lock: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return domain.ErrEditLockNotFound
	}
	return nil
}

func mapError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "duplicate key value") {
		return domain.ErrDocumentAlreadyExists
	}
	if strings.Contains(msg, "violates foreign key constraint") {
		if strings.Contains(msg, "document_type") || strings.Contains(msg, "document_profile") || strings.Contains(msg, "document_family") {
			return domain.ErrInvalidDocumentType
		}
		if strings.Contains(msg, "process_area") || strings.Contains(msg, "subject") {
			return domain.ErrInvalidCommand
		}
		return domain.ErrDocumentNotFound
	}
	if strings.Contains(msg, "document_attachments") && strings.Contains(msg, "duplicate key value") {
		return domain.ErrInvalidAttachment
	}
	return fmt.Errorf("postgres repository: %w", err)
}

func serializeDocument(document domain.Document) (tagsJSON string, metadataJSON string, effectiveAt any, expiryAt any) {
	if len(document.Tags) == 0 {
		tagsJSON = "[]"
	} else if raw, err := json.Marshal(document.Tags); err == nil {
		tagsJSON = string(raw)
	} else {
		tagsJSON = "[]"
	}

	if len(document.MetadataJSON) == 0 {
		metadataJSON = "{}"
	} else if raw, err := json.Marshal(document.MetadataJSON); err == nil {
		metadataJSON = string(raw)
	} else {
		metadataJSON = "{}"
	}

	if document.EffectiveAt != nil {
		effectiveAt = document.EffectiveAt.UTC()
	}
	if document.ExpiryAt != nil {
		expiryAt = document.ExpiryAt.UTC()
	}
	return tagsJSON, metadataJSON, effectiveAt, expiryAt
}

func nullIfEmpty(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullIfZeroInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullIfZeroInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func applyOptionalFields(doc *domain.Document, tagsJSON []byte, metadataJSON []byte, effectiveAt sql.NullTime, expiryAt sql.NullTime) {
	if len(tagsJSON) > 0 {
		var tags []string
		if err := json.Unmarshal(tagsJSON, &tags); err == nil {
			doc.Tags = tags
		}
	}
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	if len(metadataJSON) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
			doc.MetadataJSON = metadata
		}
	}
	if doc.MetadataJSON == nil {
		doc.MetadataJSON = map[string]any{}
	}
	if effectiveAt.Valid {
		t := effectiveAt.Time.UTC()
		doc.EffectiveAt = &t
	}
	if expiryAt.Valid {
		t := expiryAt.Time.UTC()
		doc.ExpiryAt = &t
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = time.Now().UTC()
	}
}

func serializeVersion(version domain.Version) (string, string, any) {
	contentSource := strings.TrimSpace(version.ContentSource)
	if contentSource == "" {
		contentSource = domain.ContentSourceNative
	}

	nativeContentJSON := "{}"
	if len(version.NativeContent) > 0 {
		if raw, err := json.Marshal(version.NativeContent); err == nil {
			nativeContentJSON = string(raw)
		}
	}

	var textContent any
	if strings.TrimSpace(version.TextContent) != "" {
		textContent = version.TextContent
	}
	return contentSource, nativeContentJSON, textContent
}

func applyVersionOptionalFields(version *domain.Version, nativeContentJSON []byte, docxStorageKey sql.NullString, pdfStorageKey sql.NullString, textContent sql.NullString, fileSizeBytes sql.NullInt64, originalFilename sql.NullString, pageCount sql.NullInt64) {
	if len(nativeContentJSON) > 0 {
		var nativeContent map[string]any
		if err := json.Unmarshal(nativeContentJSON, &nativeContent); err == nil {
			version.NativeContent = nativeContent
		}
	}
	if version.NativeContent == nil {
		version.NativeContent = map[string]any{}
	}
	if docxStorageKey.Valid {
		version.DocxStorageKey = docxStorageKey.String
	}
	if pdfStorageKey.Valid {
		version.PdfStorageKey = pdfStorageKey.String
	}
	if textContent.Valid {
		version.TextContent = textContent.String
	}
	if fileSizeBytes.Valid {
		version.FileSizeBytes = fileSizeBytes.Int64
	}
	if originalFilename.Valid {
		version.OriginalFilename = originalFilename.String
	}
	if pageCount.Valid {
		version.PageCount = int(pageCount.Int64)
	}
}
