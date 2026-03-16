package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
INSERT INTO metaldocs.documents (id, title, owner_id, classification, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	_, err := r.db.ExecContext(ctx, q,
		document.ID,
		document.Title,
		document.OwnerID,
		document.Classification,
		document.Status,
		document.CreatedAt,
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
INSERT INTO metaldocs.documents (id, title, owner_id, classification, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`
	if _, err := tx.ExecContext(ctx, insertDoc,
		document.ID,
		document.Title,
		document.OwnerID,
		document.Classification,
		document.Status,
		document.CreatedAt,
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
SELECT id, title, owner_id, classification, status, created_at
FROM metaldocs.documents
WHERE id = $1
`
	var doc domain.Document
	err := r.db.QueryRowContext(ctx, q, documentID).Scan(
		&doc.ID,
		&doc.Title,
		&doc.OwnerID,
		&doc.Classification,
		&doc.Status,
		&doc.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Document{}, domain.ErrDocumentNotFound
		}
		return domain.Document{}, fmt.Errorf("get document: %w", err)
	}
	return doc, nil
}

func (r *Repository) ListDocuments(ctx context.Context) ([]domain.Document, error) {
	const q = `
SELECT id, title, owner_id, classification, status, created_at
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
		if err := rows.Scan(
			&doc.ID,
			&doc.Title,
			&doc.OwnerID,
			&doc.Classification,
			&doc.Status,
			&doc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan document: %w", err)
		}
		out = append(out, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list documents rows: %w", err)
	}

	return out, nil
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
		return domain.ErrDocumentNotFound
	}
	return fmt.Errorf("postgres repository: %w", err)
}
