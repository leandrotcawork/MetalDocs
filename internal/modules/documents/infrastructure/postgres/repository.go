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
  id, title, document_type_code, owner_id, business_unit, department,
  classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12::jsonb, $13, $14)
`
	tagsJSON, metadataJSON, effectiveAt, expiryAt := serializeDocument(document)
	_, err := r.db.ExecContext(ctx, q,
		document.ID,
		document.Title,
		document.DocumentType,
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
  id, title, document_type_code, owner_id, business_unit, department,
  classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12::jsonb, $13, $14)
`
	tagsJSON, metadataJSON, effectiveAt, expiryAt := serializeDocument(document)
	if _, err := tx.ExecContext(ctx, insertDoc,
		document.ID,
		document.Title,
		document.DocumentType,
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
INSERT INTO metaldocs.document_versions (document_id, version_number, content, created_at)
VALUES ($1, $2, $3, $4)
`
	if _, err := tx.ExecContext(ctx, insertVersion,
		version.DocumentID,
		version.Number,
		version.Content,
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
SELECT id, title, document_type_code, owner_id, business_unit, department,
       classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
FROM metaldocs.documents
WHERE id = $1
`
	var doc domain.Document
	var tagsJSON []byte
	var metadataJSON []byte
	var effectiveAt sql.NullTime
	var expiryAt sql.NullTime
	err := r.db.QueryRowContext(ctx, q, documentID).Scan(
		&doc.ID,
		&doc.Title,
		&doc.DocumentType,
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
	applyOptionalFields(&doc, tagsJSON, metadataJSON, effectiveAt, expiryAt)
	return doc, nil
}

func (r *Repository) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	const q = `
SELECT id, title, document_type_code, owner_id, business_unit, department,
       classification, status, tags, effective_at, expiry_at, metadata_json, created_at, updated_at
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
		if err := rows.Scan(
			&doc.ID,
			&doc.Title,
			&doc.DocumentType,
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
SELECT code, name, description, review_interval_days
FROM metaldocs.document_types
ORDER BY code ASC
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
INSERT INTO metaldocs.document_versions (document_id, version_number, content, created_at)
VALUES ($1, $2, $3, $4)
`
	_, err := r.db.ExecContext(ctx, q,
		version.DocumentID,
		version.Number,
		version.Content,
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
SELECT document_id, version_number, content, created_at
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

func mapError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "duplicate key value") {
		return domain.ErrDocumentAlreadyExists
	}
	if strings.Contains(msg, "violates foreign key constraint") {
		if strings.Contains(msg, "document_type") {
			return domain.ErrInvalidDocumentType
		}
		return domain.ErrDocumentNotFound
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
