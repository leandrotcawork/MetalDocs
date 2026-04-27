package v2documents

import (
	"context"
	"database/sql"
	"fmt"

	searchdomain "metaldocs/internal/modules/search/domain"
)

type Reader struct {
	db *sql.DB
}

func NewReader(db *sql.DB) *Reader {
	return &Reader{db: db}
}

func (r *Reader) ListDocuments(ctx context.Context) ([]searchdomain.Document, error) {
	const q = `
SELECT
	d.id,
	d.name,
	COALESCE(d.status, ''),
	COALESCE(d.profile_code_snapshot, ''),
	COALESCE(d.process_area_code_snapshot, ''),
	COALESCE(d.created_by, ''),
	COALESCE(cd.code, ''),
	COALESCE(cd.sequence_num, d.revision_number, 0),
	d.created_at
FROM public.documents d
LEFT JOIN controlled_documents cd ON cd.id = d.controlled_document_id
WHERE d.archived_at IS NULL
ORDER BY d.created_at DESC
`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("v2 list documents: %w", err)
	}
	defer rows.Close()

	var out []searchdomain.Document
	for rows.Next() {
		var doc searchdomain.Document
		if err := rows.Scan(
			&doc.ID,
			&doc.Title,
			&doc.Status,
			&doc.DocumentProfile,
			&doc.ProcessArea,
			&doc.OwnerID,
			&doc.DocumentCode,
			&doc.DocumentSequence,
			&doc.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("v2 scan document: %w", err)
		}
		doc.DocumentType = doc.DocumentProfile
		out = append(out, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("v2 list documents rows: %w", err)
	}
	return out, nil
}

// ListAccessPolicies returns no policies — v2 documents use open-by-default access.
// The search service treats empty policy list as allow.
func (r *Reader) ListAccessPolicies(_ context.Context, _, _ string) ([]searchdomain.AccessPolicy, error) {
	return nil, nil
}
