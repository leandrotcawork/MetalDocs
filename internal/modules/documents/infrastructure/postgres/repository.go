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
INSERT INTO metaldocs.document_versions (document_id, version_number, content, content_hash, change_summary, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	if _, err := tx.ExecContext(ctx, insertVersion,
		version.DocumentID,
		version.Number,
		version.Content,
		version.ContentHash,
		version.ChangeSummary,
		version.CreatedAt,
	); err != nil {
		return mapError(err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx create document: %w", err)
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

func (r *Repository) ListProcessAreas(ctx context.Context) ([]domain.ProcessArea, error) {
	const q = `
SELECT code, name, description
FROM metaldocs.document_process_areas
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

func (r *Repository) ListSubjects(ctx context.Context) ([]domain.Subject, error) {
	const q = `
SELECT code, process_area_code, name, description
FROM metaldocs.document_subjects
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
INSERT INTO metaldocs.document_versions (document_id, version_number, content, content_hash, change_summary, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := r.db.ExecContext(ctx, q,
		version.DocumentID,
		version.Number,
		version.Content,
		version.ContentHash,
		version.ChangeSummary,
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
SELECT document_id, version_number, content, content_hash, change_summary, created_at
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
		if err := rows.Scan(
			&version.DocumentID,
			&version.Number,
			&version.Content,
			&version.ContentHash,
			&version.ChangeSummary,
			&version.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
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
SELECT document_id, version_number, content, content_hash, change_summary, created_at
FROM metaldocs.document_versions
WHERE document_id = $1 AND version_number = $2
`
	var version domain.Version
	if err := r.db.QueryRowContext(ctx, q, documentID, versionNumber).Scan(
		&version.DocumentID,
		&version.Number,
		&version.Content,
		&version.ContentHash,
		&version.ChangeSummary,
		&version.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.Version{}, domain.ErrVersionNotFound
		}
		return domain.Version{}, fmt.Errorf("get version: %w", err)
	}
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

func (r *Repository) CreateWorkflowApproval(ctx context.Context, approval domain.WorkflowApproval) error {
	const q = `
INSERT INTO metaldocs.workflow_approvals (
  id, document_id, requested_by, assigned_reviewer, decision_by, status,
  request_reason, decision_reason, requested_at, decided_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`
	_, err := r.db.ExecContext(ctx, q,
		approval.ID,
		approval.DocumentID,
		approval.RequestedBy,
		approval.AssignedReviewer,
		nullIfEmpty(approval.DecisionBy),
		approval.Status,
		approval.RequestReason,
		nullIfEmpty(approval.DecisionReason),
		approval.RequestedAt,
		approval.DecidedAt,
	)
	if err != nil {
		return mapError(err)
	}
	return nil
}

func (r *Repository) GetLatestWorkflowApproval(ctx context.Context, documentID string) (domain.WorkflowApproval, error) {
	const q = `
SELECT id, document_id, requested_by, assigned_reviewer, decision_by, status,
       request_reason, decision_reason, requested_at, decided_at
FROM metaldocs.workflow_approvals
WHERE document_id = $1
ORDER BY requested_at DESC
LIMIT 1
`
	var approval domain.WorkflowApproval
	var decisionBy sql.NullString
	var decisionReason sql.NullString
	var decidedAt sql.NullTime
	if err := r.db.QueryRowContext(ctx, q, documentID).Scan(
		&approval.ID,
		&approval.DocumentID,
		&approval.RequestedBy,
		&approval.AssignedReviewer,
		&decisionBy,
		&approval.Status,
		&approval.RequestReason,
		&decisionReason,
		&approval.RequestedAt,
		&decidedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.WorkflowApproval{}, domain.ErrWorkflowApprovalNotFound
		}
		return domain.WorkflowApproval{}, fmt.Errorf("get latest workflow approval: %w", err)
	}
	if decisionBy.Valid {
		approval.DecisionBy = decisionBy.String
	}
	if decisionReason.Valid {
		approval.DecisionReason = decisionReason.String
	}
	if decidedAt.Valid {
		decidedUTC := decidedAt.Time.UTC()
		approval.DecidedAt = &decidedUTC
	}
	return approval, nil
}

func (r *Repository) UpdateWorkflowApprovalDecision(ctx context.Context, approvalID, status, decisionBy, decisionReason string, decidedAt time.Time) error {
	const q = `
UPDATE metaldocs.workflow_approvals
SET status = $2, decision_by = $3, decision_reason = $4, decided_at = $5
WHERE id = $1
`
	res, err := r.db.ExecContext(ctx, q, approvalID, status, nullIfEmpty(decisionBy), nullIfEmpty(decisionReason), decidedAt.UTC())
	if err != nil {
		return fmt.Errorf("update workflow approval decision: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected update workflow approval decision: %w", err)
	}
	if affected == 0 {
		return domain.ErrWorkflowApprovalNotFound
	}
	return nil
}

func (r *Repository) SaveWorkflowApprovalState(ctx context.Context, approval domain.WorkflowApproval) error {
	const q = `
UPDATE metaldocs.workflow_approvals
SET requested_by = $2, assigned_reviewer = $3, decision_by = $4, status = $5,
    request_reason = $6, decision_reason = $7, requested_at = $8, decided_at = $9
WHERE id = $1
`
	res, err := r.db.ExecContext(ctx, q,
		approval.ID,
		approval.RequestedBy,
		approval.AssignedReviewer,
		nullIfEmpty(approval.DecisionBy),
		approval.Status,
		approval.RequestReason,
		nullIfEmpty(approval.DecisionReason),
		approval.RequestedAt.UTC(),
		approval.DecidedAt,
	)
	if err != nil {
		return fmt.Errorf("save workflow approval state: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected save workflow approval state: %w", err)
	}
	if affected == 0 {
		return domain.ErrWorkflowApprovalNotFound
	}
	return nil
}

func (r *Repository) DeleteWorkflowApproval(ctx context.Context, approvalID string) error {
	const q = `DELETE FROM metaldocs.workflow_approvals WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, approvalID)
	if err != nil {
		return fmt.Errorf("delete workflow approval: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected delete workflow approval: %w", err)
	}
	if affected == 0 {
		return domain.ErrWorkflowApprovalNotFound
	}
	return nil
}

func (r *Repository) ListWorkflowApprovals(ctx context.Context, documentID string) ([]domain.WorkflowApproval, error) {
	_, err := r.GetDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	const q = `
SELECT id, document_id, requested_by, assigned_reviewer, decision_by, status,
       request_reason, decision_reason, requested_at, decided_at
FROM metaldocs.workflow_approvals
WHERE document_id = $1
ORDER BY requested_at ASC
`
	rows, err := r.db.QueryContext(ctx, q, documentID)
	if err != nil {
		return nil, fmt.Errorf("list workflow approvals: %w", err)
	}
	defer rows.Close()

	var out []domain.WorkflowApproval
	for rows.Next() {
		var approval domain.WorkflowApproval
		var decisionBy sql.NullString
		var decisionReason sql.NullString
		var decidedAt sql.NullTime
		if err := rows.Scan(
			&approval.ID,
			&approval.DocumentID,
			&approval.RequestedBy,
			&approval.AssignedReviewer,
			&decisionBy,
			&approval.Status,
			&approval.RequestReason,
			&decisionReason,
			&approval.RequestedAt,
			&decidedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workflow approval: %w", err)
		}
		if decisionBy.Valid {
			approval.DecisionBy = decisionBy.String
		}
		if decisionReason.Valid {
			approval.DecisionReason = decisionReason.String
		}
		if decidedAt.Valid {
			decidedUTC := decidedAt.Time.UTC()
			approval.DecidedAt = &decidedUTC
		}
		out = append(out, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list workflow approvals rows: %w", err)
	}
	return out, nil
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
