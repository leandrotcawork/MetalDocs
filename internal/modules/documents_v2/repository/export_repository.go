package repository

import (
	"context"
	"database/sql"
	"errors"

	"metaldocs/internal/modules/documents_v2/domain"
)

func (r *Repository) InsertExport(ctx context.Context, e *domain.Export) (*domain.Export, error) {
	var inserted domain.Export
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO document_exports
		     (document_id, revision_id, composite_hash, storage_key, size_bytes, paper_size, landscape, docgen_v2_ver)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (document_id, composite_hash) DO NOTHING
		 RETURNING id, document_id, revision_id, composite_hash, storage_key, size_bytes, paper_size, landscape, docgen_v2_ver`,
		e.DocumentID, e.RevisionID, e.CompositeHash, e.StorageKey,
		e.SizeBytes, e.PaperSize, e.Landscape, e.DocgenV2Ver,
	).Scan(
		&inserted.ID, &inserted.DocumentID, &inserted.RevisionID, &inserted.CompositeHash,
		&inserted.StorageKey, &inserted.SizeBytes, &inserted.PaperSize, &inserted.Landscape, &inserted.DocgenV2Ver,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return r.GetExportByHash(ctx, e.DocumentID, e.CompositeHash)
	}
	if err != nil {
		return nil, err
	}
	return &inserted, nil
}

func (r *Repository) GetExportByHash(ctx context.Context, documentID string, compositeHash []byte) (*domain.Export, error) {
	var e domain.Export
	err := r.db.QueryRowContext(ctx,
		`SELECT id, document_id, revision_id, composite_hash, storage_key, size_bytes, paper_size, landscape, docgen_v2_ver
		 FROM document_exports
		 WHERE document_id = $1 AND composite_hash = $2`,
		documentID, compositeHash,
	).Scan(
		&e.ID, &e.DocumentID, &e.RevisionID, &e.CompositeHash,
		&e.StorageKey, &e.SizeBytes, &e.PaperSize, &e.Landscape, &e.DocgenV2Ver,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
