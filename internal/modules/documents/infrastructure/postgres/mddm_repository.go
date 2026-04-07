package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

type MDDMRepository struct {
	db *sql.DB
}

func NewMDDMRepository(db *sql.DB) *MDDMRepository {
	return &MDDMRepository{db: db}
}

type InsertDraftParams struct {
	DocumentID    string
	VersionNumber int
	RevisionLabel string
	ContentBlocks json.RawMessage
	ContentHash   string
	TemplateRef   json.RawMessage
	CreatedBy     string
}

func (r *MDDMRepository) InsertDraft(ctx context.Context, p InsertDraftParams) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO metaldocs.document_versions_mddm
		  (document_id, version_number, revision_label, status, content_blocks, content_hash, template_ref, created_by)
		VALUES ($1, $2, $3, 'draft', $4, $5, $6, $7)
		RETURNING id
	`, p.DocumentID, p.VersionNumber, p.RevisionLabel, p.ContentBlocks, p.ContentHash, p.TemplateRef, p.CreatedBy).Scan(&id)
	return id, err
}

type DocumentVersion struct {
	ID            uuid.UUID
	DocumentID    string
	VersionNumber int
	RevisionLabel string
	Status        string
	ContentBlocks json.RawMessage
	DocxBytes     []byte
	TemplateRef   json.RawMessage
	ContentHash   string
	RevisionDiff  json.RawMessage
}

func (r *MDDMRepository) GetCurrentReleased(ctx context.Context, documentID string) (*DocumentVersion, error) {
	var v DocumentVersion
	err := r.db.QueryRowContext(ctx, `
		SELECT id, document_id, version_number, revision_label, status, content_blocks, docx_bytes, template_ref, content_hash, revision_diff
		FROM metaldocs.document_versions_mddm
		WHERE document_id = $1 AND status = 'released'
	`, documentID).Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.RevisionLabel, &v.Status, &v.ContentBlocks, &v.DocxBytes, &v.TemplateRef, &v.ContentHash, &v.RevisionDiff)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

func (r *MDDMRepository) GetActiveDraft(ctx context.Context, documentID string) (*DocumentVersion, error) {
	var v DocumentVersion
	err := r.db.QueryRowContext(ctx, `
		SELECT id, document_id, version_number, revision_label, status, content_blocks, docx_bytes, template_ref, content_hash, revision_diff
		FROM metaldocs.document_versions_mddm
		WHERE document_id = $1 AND status IN ('draft', 'pending_approval')
	`, documentID).Scan(&v.ID, &v.DocumentID, &v.VersionNumber, &v.RevisionLabel, &v.Status, &v.ContentBlocks, &v.DocxBytes, &v.TemplateRef, &v.ContentHash, &v.RevisionDiff)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

func (r *MDDMRepository) UpdateDraftContent(ctx context.Context, id uuid.UUID, content json.RawMessage, hash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE metaldocs.document_versions_mddm
		SET content_blocks = $1, content_hash = $2
		WHERE id = $3 AND status = 'draft'
	`, content, hash, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
