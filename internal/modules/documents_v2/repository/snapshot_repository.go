package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"metaldocs/internal/modules/documents_v2/domain"
)

// SnapshotRepository reads and writes the template snapshot columns on documents.
type SnapshotRepository struct {
	db     *sql.DB
	schema string // optional schema prefix; empty = bare table name
}

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewSnapshotRepository creates a SnapshotRepository using bare table names.
// In tests, use NewSnapshotRepositoryWithSchema to point at the isolated test schema.
func NewSnapshotRepository(db *sql.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

// NewSnapshotRepositoryWithSchema creates a SnapshotRepository that qualifies
// table names with the given schema. Used by integration tests.
func NewSnapshotRepositoryWithSchema(db *sql.DB, schema string) *SnapshotRepository {
	return &SnapshotRepository{db: db, schema: schema}
}

func (r *SnapshotRepository) table(name string) string {
	if r.schema == "" {
		return name
	}
	return fmt.Sprintf("%q.%q", r.schema, name)
}

// WriteSnapshot updates the snapshot columns for a document.
func (r *SnapshotRepository) WriteSnapshot(ctx context.Context, tenantID, docID string, s domain.TemplateSnapshot) error {
	h := s.Hashes()
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		   SET placeholder_schema_snapshot    = $1,
		       placeholder_schema_hash        = $2,
		       composition_config_snapshot    = $3,
		       composition_config_hash        = $4,
		       body_docx_snapshot_s3_key      = $5,
		       body_docx_hash                 = $6
		 WHERE tenant_id = $7::uuid AND id = $8::uuid`, r.table("documents")),
		s.PlaceholderSchemaJSON, h.PlaceholderSchemaHash,
		s.CompositionJSON, h.CompositionHash,
		s.BodyDocxS3Key, h.BodyDocxHash,
		tenantID, docID,
	)
	return err
}

// ReadSnapshot reads the snapshot columns for a document.
func (r *SnapshotRepository) ReadSnapshot(ctx context.Context, tenantID, docID string) (domain.TemplateSnapshot, error) {
	s, _, err := r.readSnapshot(ctx, r.db, tenantID, docID)
	return s, err
}

// ReadSnapshotWithFreezeAt reads snapshot columns and values_frozen_at for idempotency checks.
func (r *SnapshotRepository) ReadSnapshotWithFreezeAt(ctx context.Context, tenantID, docID string, q ...DBTX) (domain.TemplateSnapshot, *time.Time, error) {
	exec := DBTX(r.db)
	if len(q) > 0 && q[0] != nil {
		exec = q[0]
	}
	return r.readSnapshot(ctx, exec, tenantID, docID)
}

func (r *SnapshotRepository) readSnapshot(ctx context.Context, exec DBTX, tenantID, docID string) (domain.TemplateSnapshot, *time.Time, error) {
	var s domain.TemplateSnapshot
	var valuesFrozenAt *time.Time
	err := exec.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT placeholder_schema_snapshot,
		       composition_config_snapshot,
		       coalesce(body_docx_snapshot_s3_key, ''),
		       values_frozen_at
		  FROM %s
		 WHERE tenant_id = $1::uuid AND id = $2::uuid`, r.table("documents")),
		tenantID, docID,
	).Scan(
		&s.PlaceholderSchemaJSON,
		&s.CompositionJSON,
		&s.BodyDocxS3Key,
		&valuesFrozenAt,
	)
	return s, valuesFrozenAt, err
}

func (r *SnapshotRepository) WriteFreeze(ctx context.Context, tenant, docID string, valuesHash []byte, frozenAt time.Time, q ...DBTX) error {
	exec := DBTX(r.db)
	if len(q) > 0 && q[0] != nil {
		exec = q[0]
	}
	_, err := exec.ExecContext(ctx, fmt.Sprintf(`
        UPDATE %s
           SET values_hash=$1, values_frozen_at=$2
         WHERE tenant_id=$3 AND id=$4`, r.table("documents")),
		valuesHash, frozenAt, tenant, docID)
	return err
}

// WriteFinalDocx persists the fanout output pointer and content hash onto a document.
func (r *SnapshotRepository) WriteFinalDocx(ctx context.Context, tenant, docID, s3Key string, contentHash []byte, q ...DBTX) error {
	exec := DBTX(r.db)
	if len(q) > 0 && q[0] != nil {
		exec = q[0]
	}
	_, err := exec.ExecContext(ctx, fmt.Sprintf(`
        UPDATE %s
           SET final_docx_s3_key=$1, content_hash=$2
         WHERE tenant_id=$3::uuid AND id=$4::uuid`, r.table("documents")),
		s3Key, contentHash, tenant, docID)
	return err
}

// WritePDF persists the rendered PDF pointer, hash, and generation timestamp.
func (r *SnapshotRepository) WritePDF(ctx context.Context, tenant, docID, s3Key string, pdfHash []byte, generatedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
        UPDATE %s
           SET final_pdf_s3_key=$1, pdf_hash=$2, pdf_generated_at=$3
         WHERE tenant_id=$4::uuid AND id=$5::uuid`, r.table("documents")),
		s3Key, pdfHash, generatedAt, tenant, docID)
	return err
}

// AppendReconstruction appends a forensic attempt entry onto documents.reconstruction_attempts.
// Never touches final_docx_s3_key or content_hash.
func (r *SnapshotRepository) AppendReconstruction(ctx context.Context, tenant, docID string, entry []byte) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
        UPDATE %s
           SET reconstruction_attempts = reconstruction_attempts || $1::jsonb
         WHERE tenant_id=$2::uuid AND id=$3::uuid`, r.table("documents")),
		entry, tenant, docID)
	return err
}
